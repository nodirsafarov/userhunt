// Package checker implements userhunt's concurrent username probe engine.
//
// Each Checker reuses a single http.Client (with keep-alive, HTTP/2, optional
// proxy) and dispatches probes through a worker pool. Probes use the platform
// definitions from internal/platforms and stream results back on a channel as
// soon as they complete, so callers can render progress in real time.
package checker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nodirsafarov/userhunt/internal/platforms"
)

// Status describes the outcome of probing a single platform.
type Status string

const (
	StatusFound    Status = "found"
	StatusNotFound Status = "not_found"
	StatusError    Status = "error"
	StatusTimeout  Status = "timeout"
)

// Result is the outcome of probing one Platform for one username.
type Result struct {
	Platform platforms.Platform
	Username string
	Status   Status
	URL      string
	HTTPCode int
	Duration time.Duration
	Err      string
}

// Options configures a Checker. Zero values fall back to sensible defaults.
type Options struct {
	Timeout     time.Duration
	Concurrency int
	Retries     int
	UserAgent   string
	Proxy       string
}

// Checker performs concurrent username probes against a platform list.
type Checker struct {
	client      *http.Client
	concurrency int
	retries     int
	userAgent   string
	rng         *rand.Rand
	mu          sync.Mutex
}

const (
	defaultTimeout     = 15 * time.Second
	defaultConcurrency = 50
	defaultRetries     = 1
)

var defaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0",
}

// New constructs a Checker with the supplied Options.
func New(opts Options) (*Checker, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = defaultConcurrency
	}
	if opts.Retries < 0 {
		opts.Retries = defaultRetries
	}

	transport := &http.Transport{
		MaxIdleConns:        opts.Concurrency * 2,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if opts.Proxy != "" {
		proxyURL, err := url.Parse(opts.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &Checker{
		client: &http.Client{
			Timeout:   opts.Timeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		concurrency: opts.Concurrency,
		retries:     opts.Retries,
		userAgent:   opts.UserAgent,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// Run dispatches concurrent probes for every platform in list against username
// and streams results on the returned channel. The channel closes when all
// probes finish (or ctx is canceled).
func (c *Checker) Run(ctx context.Context, username string, list []platforms.Platform) <-chan Result {
	out := make(chan Result, c.concurrency)
	jobs := make(chan platforms.Platform, c.concurrency)

	var wg sync.WaitGroup
	for i := 0; i < c.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				out <- c.probe(ctx, username, p)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, p := range list {
			select {
			case <-ctx.Done():
				return
			case jobs <- p:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func (c *Checker) probe(ctx context.Context, username string, p platforms.Platform) Result {
	start := time.Now()
	target := p.BuildURL(username)
	res := Result{
		Platform: p,
		Username: username,
		URL:      target,
	}

	var resp *http.Response
	var body []byte
	var lastErr error

	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * 200 * time.Millisecond
			select {
			case <-ctx.Done():
				res.Status = StatusTimeout
				res.Duration = time.Since(start)
				res.Err = ctx.Err().Error()
				return res
			case <-time.After(backoff):
			}
		}

		resp, body, lastErr = c.fetch(ctx, target)
		if lastErr == nil {
			break
		}
	}

	res.Duration = time.Since(start)
	if lastErr != nil {
		if errors.Is(lastErr, context.DeadlineExceeded) || isTimeout(lastErr) {
			res.Status = StatusTimeout
		} else {
			res.Status = StatusError
		}
		res.Err = lastErr.Error()
		return res
	}

	res.HTTPCode = resp.StatusCode
	finalURL := target
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	res.Status = decide(p, resp.StatusCode, body, finalURL)
	if res.Status == StatusFound {
		res.URL = finalURL
	}
	return res
}

func (c *Checker) fetch(ctx context.Context, target string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", c.pickUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return resp, nil, err
	}
	return resp, body, nil
}

func (c *Checker) pickUserAgent() string {
	if c.userAgent != "" {
		return c.userAgent
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return defaultUserAgents[c.rng.Intn(len(defaultUserAgents))]
}

func decide(p platforms.Platform, code int, body []byte, finalURL string) Status {
	for _, marker := range p.NotExistsFinalURL {
		if marker != "" && strings.Contains(finalURL, marker) {
			return StatusNotFound
		}
	}

	switch p.CheckType {
	case platforms.CheckStatus:
		if code == http.StatusOK {
			return StatusFound
		}
		if code == http.StatusNotFound || code == http.StatusGone {
			return StatusNotFound
		}
		return StatusError

	case platforms.CheckContent:
		if code >= 500 {
			return StatusError
		}
		text := string(body)
		for _, marker := range p.ExistsContent {
			if strings.Contains(text, marker) {
				return StatusFound
			}
		}
		for _, marker := range p.NotExistsContent {
			if strings.Contains(text, marker) {
				return StatusNotFound
			}
		}
		if len(p.ExistsContent) > 0 {
			return StatusNotFound
		}
		if code == http.StatusOK {
			return StatusFound
		}
		if code == http.StatusNotFound {
			return StatusNotFound
		}
		return StatusError
	}
	return StatusError
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	type timeoutErr interface{ Timeout() bool }
	var te timeoutErr
	if errors.As(err, &te) {
		return te.Timeout()
	}
	return strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline")
}
