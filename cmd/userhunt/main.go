// Command userhunt is a fast OSINT username enumerator.
//
//	userhunt <username> [flags]
//
// See `userhunt --help` for the full flag reference.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/nodirsafarov/userhunt/internal/checker"
	"github.com/nodirsafarov/userhunt/internal/output"
	"github.com/nodirsafarov/userhunt/internal/platforms"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type runOpts struct {
	timeout      time.Duration
	concurrency  int
	retries      int
	userAgent    string
	proxy        string
	output       string
	format       string
	onlyFound    bool
	noColor      bool
	noBanner     bool
	category     string
	listSites    bool
	includeNSFW  bool
	silent       bool
	exitNonzero  bool
}

func main() {
	opts := &runOpts{}
	cmd := &cobra.Command{
		Use:   "userhunt <username> [username...]",
		Short: "Fast OSINT username enumerator across 100+ platforms",
		Long: `userhunt scans 100+ websites in parallel to find accounts for a given username.

Examples:
  userhunt elonmusk
  userhunt elonmusk -o report.json
  userhunt elonmusk --format md -o report.md
  userhunt user1 user2 user3 --only-found
  userhunt elonmusk --category tech --concurrency 100
  userhunt --list`,
		Version: fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		Args:    cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return run(c.Context(), opts, args)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().DurationVarP(&opts.timeout, "timeout", "t", 15*time.Second, "Per-request timeout")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "c", 50, "Number of parallel workers")
	cmd.Flags().IntVar(&opts.retries, "retries", 1, "Retry attempts per platform")
	cmd.Flags().StringVar(&opts.userAgent, "user-agent", "", "Custom User-Agent (default: rotating real-browser UAs)")
	cmd.Flags().StringVar(&opts.proxy, "proxy", "", "HTTP(S) proxy URL (e.g. http://127.0.0.1:8080)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Write report to file (extension inferred if --format omitted)")
	cmd.Flags().StringVarP(&opts.format, "format", "f", "", "Report format: json | csv | md")
	cmd.Flags().BoolVar(&opts.onlyFound, "only-found", false, "Suppress not-found / error lines")
	cmd.Flags().BoolVar(&opts.noColor, "no-color", false, "Disable colored output")
	cmd.Flags().BoolVar(&opts.noBanner, "no-banner", false, "Hide the ASCII banner")
	cmd.Flags().StringVar(&opts.category, "category", "", "Restrict to one category (use --list to see them)")
	cmd.Flags().BoolVar(&opts.listSites, "list", false, "List available platforms / categories and exit")
	cmd.Flags().BoolVar(&opts.includeNSFW, "include-nsfw", false, "Include NSFW platforms")
	cmd.Flags().BoolVarP(&opts.silent, "silent", "s", false, "Suppress progress bar and banner")
	cmd.Flags().BoolVar(&opts.exitNonzero, "fail-if-not-found", false, "Exit with code 2 if zero accounts found")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", color.New(color.FgRed, color.Bold).Sprint("error:"), err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opts *runOpts, args []string) error {
	if opts.noColor {
		color.NoColor = true
	}

	all, err := platforms.Load()
	if err != nil {
		return err
	}

	if opts.listSites {
		listPlatforms(os.Stdout, all)
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("at least one username is required (use --list to see platforms)")
	}

	usernames := dedupe(args)
	for _, u := range usernames {
		if err := validateUsername(u); err != nil {
			return err
		}
	}

	list := platforms.Filter(all, opts.category, opts.includeNSFW)
	if len(list) == 0 {
		return fmt.Errorf("no platforms match category %q", opts.category)
	}

	chk, err := checker.New(checker.Options{
		Timeout:     opts.timeout,
		Concurrency: opts.concurrency,
		Retries:     opts.retries,
		UserAgent:   opts.userAgent,
		Proxy:       opts.proxy,
	})
	if err != nil {
		return err
	}

	totalFound := 0
	for _, username := range usernames {
		found, err := scanOne(ctx, chk, list, username, opts)
		if err != nil {
			return err
		}
		totalFound += found
	}

	if opts.exitNonzero && totalFound == 0 {
		os.Exit(2)
	}
	return nil
}

func scanOne(ctx context.Context, chk *checker.Checker, list []platforms.Platform, username string, opts *runOpts) (int, error) {
	if !opts.silent && !opts.noBanner {
		output.PrintBanner(os.Stderr)
	}
	if !opts.silent {
		fmt.Fprintf(os.Stderr, "  Hunting %s across %d platforms...\n\n",
			color.New(color.FgCyan, color.Bold).Sprintf("@%s", username), len(list))
	}

	showBar := !opts.silent && opts.onlyFound
	var bar *progressbar.ProgressBar
	if showBar {
		bar = progressbar.NewOptions(len(list),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionSetDescription("scanning"),
			progressbar.OptionShowCount(),
			progressbar.OptionThrottle(80*time.Millisecond),
			progressbar.OptionSetWidth(30),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSpinnerType(14),
		)
	}

	start := time.Now()
	results := make([]checker.Result, 0, len(list))
	for r := range chk.Run(ctx, username, list) {
		results = append(results, r)
		if showBar {
			if r.Status == checker.StatusFound {
				_ = bar.Clear()
				output.PrintResult(os.Stderr, r, opts.onlyFound)
			}
			_ = bar.Add(1)
		} else if !opts.silent {
			output.PrintResult(os.Stderr, r, opts.onlyFound)
		}
	}
	runtime := time.Since(start)
	if bar != nil {
		_ = bar.Finish()
	}

	recs, sum := output.FromResults(username, results, runtime)
	if !opts.silent {
		output.PrintSummary(os.Stderr, sum)
	}

	if opts.output != "" {
		path, format := resolveOutputPath(opts.output, opts.format, username)
		if err := output.SaveFile(path, format, recs, sum); err != nil {
			return sum.Found, fmt.Errorf("write %s: %w", path, err)
		}
		if !opts.silent {
			fmt.Fprintf(os.Stderr, "  %s %s\n",
				color.New(color.FgCyan, color.Bold).Sprint("Saved:"), path)
		}
	}

	return sum.Found, nil
}

func resolveOutputPath(out, format, username string) (string, string) {
	if format == "" {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(out), "."))
		switch ext {
		case "json", "csv", "md", "markdown":
			format = ext
		default:
			format = "json"
		}
	}
	out = strings.ReplaceAll(out, "{user}", username)
	out = strings.ReplaceAll(out, "{username}", username)
	if filepath.Ext(out) == "" {
		out = fmt.Sprintf("%s.%s", out, format)
	}
	return out, format
}

func listPlatforms(w *os.File, all []platforms.Platform) {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Fprintf(w, "%s  %d platforms\n\n", cyan("userhunt:"), len(all))
	fmt.Fprintln(w, bold("Categories:"))
	for _, c := range platforms.Categories(all) {
		fmt.Fprintf(w, "  - %s\n", c)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, bold("Platforms:"))
	for _, p := range all {
		nsfw := ""
		if p.NSFW {
			nsfw = dim(" [nsfw]")
		}
		fmt.Fprintf(w, "  %-30s %s%s\n", p.Name, dim(p.Category), nsfw)
	}
}

func validateUsername(u string) error {
	if u == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if strings.ContainsAny(u, " \t\n\r/?#%&=") {
		return fmt.Errorf("invalid character in username %q", u)
	}
	if len(u) > 64 {
		return fmt.Errorf("username too long (max 64 chars): %q", u)
	}
	return nil
}

func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
