package platforms

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	all, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(all) < 50 {
		t.Fatalf("expected at least 50 platforms, got %d", len(all))
	}

	seen := make(map[string]bool)
	for _, p := range all {
		if seen[p.Name] {
			t.Errorf("duplicate platform name: %s", p.Name)
		}
		seen[p.Name] = true

		if !strings.Contains(p.URL, "{}") {
			t.Errorf("platform %s: URL missing {} placeholder: %s", p.Name, p.URL)
		}
		if !strings.HasPrefix(p.URL, "http://") && !strings.HasPrefix(p.URL, "https://") {
			t.Errorf("platform %s: URL must start with http(s)://, got %s", p.Name, p.URL)
		}
		if p.Category == "" {
			t.Errorf("platform %s: missing category", p.Name)
		}
	}
}

func TestBuildURL(t *testing.T) {
	cases := []struct {
		template string
		username string
		want     string
	}{
		{"https://github.com/{}", "octocat", "https://github.com/octocat"},
		{"https://{}.wordpress.com", "matt", "https://matt.wordpress.com"},
		{"https://news.ycombinator.com/user?id={}", "pg", "https://news.ycombinator.com/user?id=pg"},
	}
	for _, tc := range cases {
		p := Platform{URL: tc.template}
		got := p.BuildURL(tc.username)
		if got != tc.want {
			t.Errorf("BuildURL(%q, %q) = %q, want %q", tc.template, tc.username, got, tc.want)
		}
	}
}

func TestFilter(t *testing.T) {
	all, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tech := Filter(all, "tech", false)
	if len(tech) == 0 {
		t.Fatal("expected at least one tech platform")
	}
	for _, p := range tech {
		if p.Category != "tech" {
			t.Errorf("tech filter returned %s with category %s", p.Name, p.Category)
		}
	}

	none := Filter(all, "nonexistent-xyz", false)
	if len(none) != 0 {
		t.Errorf("expected 0 platforms for nonexistent category, got %d", len(none))
	}
}

func TestCategories(t *testing.T) {
	all, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cats := Categories(all)
	if len(cats) < 3 {
		t.Errorf("expected at least 3 categories, got %d: %v", len(cats), cats)
	}
}
