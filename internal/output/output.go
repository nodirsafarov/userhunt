// Package output renders userhunt results in human and machine formats.
//
// Terminal output is colored and column-aligned. Machine formats (JSON, CSV,
// Markdown) write deterministic, schema-stable documents suitable for piping
// into other tools or reports.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/nodirsafarov/userhunt/internal/checker"
)

// Summary aggregates the totals at the end of a probe run.
type Summary struct {
	Username  string        `json:"username"`
	Total     int           `json:"total"`
	Found     int           `json:"found"`
	NotFound  int           `json:"not_found"`
	Errors    int           `json:"errors"`
	Duration  time.Duration `json:"duration_ns"`
	Generated time.Time     `json:"generated_at"`
}

// Record is a JSON/CSV/Markdown-friendly view of a checker.Result.
type Record struct {
	Platform string `json:"platform"`
	Category string `json:"category"`
	Status   string `json:"status"`
	URL      string `json:"url"`
	HTTPCode int    `json:"http_code,omitempty"`
	Duration int64  `json:"duration_ms"`
	Error    string `json:"error,omitempty"`
}

// FromResults converts checker results into stable Records and computes the
// run Summary.
func FromResults(username string, results []checker.Result, runtime time.Duration) ([]Record, Summary) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Status != results[j].Status {
			return statusRank(results[i].Status) < statusRank(results[j].Status)
		}
		return strings.ToLower(results[i].Platform.Name) < strings.ToLower(results[j].Platform.Name)
	})

	recs := make([]Record, 0, len(results))
	sum := Summary{
		Username:  username,
		Total:     len(results),
		Duration:  runtime,
		Generated: time.Now().UTC(),
	}
	for _, r := range results {
		recs = append(recs, Record{
			Platform: r.Platform.Name,
			Category: r.Platform.Category,
			Status:   string(r.Status),
			URL:      r.URL,
			HTTPCode: r.HTTPCode,
			Duration: r.Duration.Milliseconds(),
			Error:    r.Err,
		})
		switch r.Status {
		case checker.StatusFound:
			sum.Found++
		case checker.StatusNotFound:
			sum.NotFound++
		default:
			sum.Errors++
		}
	}
	return recs, sum
}

// PrintBanner prints the userhunt ASCII banner to stderr.
func PrintBanner(w io.Writer) {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	fmt.Fprintln(w, cyan(`
                            ╔╗   ╔╗
                            ║║   ║║
╔╗ ╔╗╔══╗╔══╗╔══╗╔═╦╗╔╗╔╗╔═╗║╚═╗ ║║
║║ ║║║══╣║╔╗║║╔╗║║╔╣╚╝║║║║╔╗╠══╗║ ╚╝
║╚═╝║╠══║║║═╣║║║║║║║║║║║╠╝╠╣║═╣║ ╔╗
╚═╗╔╝╚══╝╚══╝╚╝╚╝╚╝╚╝╚╝╚╝╚╝╚══╝╚ ╚╝
  ║║
  ╚╝   `))
	fmt.Fprintln(w, dim("           Fast OSINT username enumerator — by @nodirsafarov"))
	fmt.Fprintln(w)
}

// PrintResult writes a single live result line to stderr.
func PrintResult(w io.Writer, r checker.Result, onlyFound bool) {
	if onlyFound && r.Status != checker.StatusFound {
		return
	}

	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	red := color.New(color.FgRed, color.Faint).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	const platformCol = 26
	runes := []rune(r.Platform.Name)
	name := r.Platform.Name
	if len(runes) > platformCol {
		name = string(runes[:platformCol-1]) + "…"
	}
	pad := platformCol - utf8.RuneCountInString(name)
	if pad < 0 {
		pad = 0
	}
	padded := name + strings.Repeat(" ", pad)

	switch r.Status {
	case checker.StatusFound:
		fmt.Fprintf(w, "  %s  %s  %s\n", green("[+]"), bold(padded), r.URL)
	case checker.StatusNotFound:
		if !onlyFound {
			fmt.Fprintf(w, "  %s  %s  %s\n", red("[-]"), dim(padded), dim("not found"))
		}
	case checker.StatusTimeout:
		if !onlyFound {
			fmt.Fprintf(w, "  %s  %s  %s\n", yellow("[!]"), padded, dim("timeout"))
		}
	default:
		if !onlyFound {
			msg := r.Err
			if msg == "" {
				msg = "error"
			}
			fmt.Fprintf(w, "  %s  %s  %s\n", yellow("[!]"), padded, dim(msg))
		}
	}
}

// PrintSummary writes the closing summary block to stderr.
func PrintSummary(w io.Writer, sum Summary) {
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Fprintln(w)
	fmt.Fprintln(w, cyan("──────────────────────────────────────────────"))
	fmt.Fprintf(w, "  %s  %s\n", cyan("Target:"), sum.Username)
	fmt.Fprintf(w, "  %s   %s / %d platforms\n", cyan("Found:"), green(fmt.Sprintf("%d", sum.Found)), sum.Total)
	fmt.Fprintf(w, "  %s   %d\n", cyan("Errors:"), sum.Errors)
	fmt.Fprintf(w, "  %s    %s\n", cyan("Time:"), dim(sum.Duration.Round(time.Millisecond).String()))
	fmt.Fprintln(w, cyan("──────────────────────────────────────────────"))
}

// WriteJSON serializes records+summary as pretty-printed JSON.
func WriteJSON(w io.Writer, recs []Record, sum Summary) error {
	doc := struct {
		Summary Summary  `json:"summary"`
		Results []Record `json:"results"`
	}{sum, recs}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// WriteCSV writes records as comma-separated values with a header row.
func WriteCSV(w io.Writer, recs []Record) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"platform", "category", "status", "url", "http_code", "duration_ms", "error"}); err != nil {
		return err
	}
	for _, r := range recs {
		row := []string{
			r.Platform,
			r.Category,
			r.Status,
			r.URL,
			fmt.Sprintf("%d", r.HTTPCode),
			fmt.Sprintf("%d", r.Duration),
			r.Error,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteMarkdown writes a human-readable Markdown report grouped by status.
func WriteMarkdown(w io.Writer, recs []Record, sum Summary) error {
	fmt.Fprintf(w, "# userhunt report — `%s`\n\n", sum.Username)
	fmt.Fprintf(w, "Generated: `%s`\n\n", sum.Generated.Format(time.RFC3339))
	fmt.Fprintln(w, "| Metric | Value |")
	fmt.Fprintln(w, "|---|---|")
	fmt.Fprintf(w, "| Target | `%s` |\n", sum.Username)
	fmt.Fprintf(w, "| Platforms scanned | %d |\n", sum.Total)
	fmt.Fprintf(w, "| Accounts found | **%d** |\n", sum.Found)
	fmt.Fprintf(w, "| Not found | %d |\n", sum.NotFound)
	fmt.Fprintf(w, "| Errors | %d |\n", sum.Errors)
	fmt.Fprintf(w, "| Duration | %s |\n\n", sum.Duration.Round(time.Millisecond))

	fmt.Fprintln(w, "## ✅ Found accounts")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Platform | Category | URL |")
	fmt.Fprintln(w, "|---|---|---|")
	for _, r := range recs {
		if r.Status == string(checker.StatusFound) {
			fmt.Fprintf(w, "| %s | %s | <%s> |\n", r.Platform, r.Category, r.URL)
		}
	}

	hasErr := false
	for _, r := range recs {
		if r.Status == string(checker.StatusError) || r.Status == string(checker.StatusTimeout) {
			hasErr = true
			break
		}
	}
	if hasErr {
		fmt.Fprintln(w, "\n## ⚠️  Errors / timeouts")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| Platform | Status | Error |")
		fmt.Fprintln(w, "|---|---|---|")
		for _, r := range recs {
			if r.Status == string(checker.StatusError) || r.Status == string(checker.StatusTimeout) {
				fmt.Fprintf(w, "| %s | %s | %s |\n", r.Platform, r.Status, r.Error)
			}
		}
	}

	return nil
}

// SaveFile writes content to path using the matching format. fmt may be "json",
// "csv" or "md".
func SaveFile(path, format string, recs []Record, sum Summary) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	switch strings.ToLower(format) {
	case "json":
		return WriteJSON(f, recs, sum)
	case "csv":
		return WriteCSV(f, recs)
	case "md", "markdown":
		return WriteMarkdown(f, recs, sum)
	default:
		return fmt.Errorf("unknown format %q (want json, csv, md)", format)
	}
}

func statusRank(s checker.Status) int {
	switch s {
	case checker.StatusFound:
		return 0
	case checker.StatusNotFound:
		return 2
	case checker.StatusTimeout:
		return 3
	case checker.StatusError:
		return 4
	}
	return 5
}
