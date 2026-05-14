package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nodirsafarov/userhunt/internal/platforms"
)

func TestDecideStatus(t *testing.T) {
	p := platforms.Platform{CheckType: platforms.CheckStatus}
	cases := []struct {
		code int
		want Status
	}{
		{200, StatusFound},
		{404, StatusNotFound},
		{410, StatusNotFound},
		{500, StatusError},
		{403, StatusError},
	}
	for _, tc := range cases {
		if got := decide(p, tc.code, nil, ""); got != tc.want {
			t.Errorf("code %d: got %s, want %s", tc.code, got, tc.want)
		}
	}
}

func TestDecideContentNotExists(t *testing.T) {
	p := platforms.Platform{
		CheckType:        platforms.CheckContent,
		NotExistsContent: []string{"No such user"},
	}
	if got := decide(p, 200, []byte("page with No such user marker"), ""); got != StatusNotFound {
		t.Errorf("expected NotFound, got %s", got)
	}
	if got := decide(p, 200, []byte("profile content"), ""); got != StatusFound {
		t.Errorf("expected Found, got %s", got)
	}
}

func TestDecideContentExists(t *testing.T) {
	p := platforms.Platform{
		CheckType:     platforms.CheckContent,
		ExistsContent: []string{"profile_photo"},
	}
	if got := decide(p, 200, []byte("html with profile_photo here"), ""); got != StatusFound {
		t.Errorf("expected Found, got %s", got)
	}
	if got := decide(p, 200, []byte("nothing matching"), ""); got != StatusNotFound {
		t.Errorf("expected NotFound (no marker match), got %s", got)
	}
}

func TestDecideFinalURLOverride(t *testing.T) {
	p := platforms.Platform{
		CheckType:         platforms.CheckStatus,
		NotExistsFinalURL: []string{"/login", "/sorry"},
	}
	if got := decide(p, 200, nil, "https://example.com/login?return=/x"); got != StatusNotFound {
		t.Errorf("expected NotFound from redirect to /login, got %s", got)
	}
	if got := decide(p, 200, nil, "https://example.com/profile/x"); got != StatusFound {
		t.Errorf("expected Found when final URL has no marker, got %s", got)
	}
}

func TestRunEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/found/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/notfound/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	list := []platforms.Platform{
		{Name: "fake-found", URL: srv.URL + "/found/{}", Category: "test", CheckType: platforms.CheckStatus},
		{Name: "fake-notfound", URL: srv.URL + "/notfound/{}", Category: "test", CheckType: platforms.CheckStatus},
	}

	c, err := New(Options{Timeout: 5 * time.Second, Concurrency: 2})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	got := map[string]Status{}
	for r := range c.Run(ctx, "anyname", list) {
		got[r.Platform.Name] = r.Status
	}
	if got["fake-found"] != StatusFound {
		t.Errorf("fake-found: got %s, want found", got["fake-found"])
	}
	if got["fake-notfound"] != StatusNotFound {
		t.Errorf("fake-notfound: got %s, want not_found", got["fake-notfound"])
	}
}
