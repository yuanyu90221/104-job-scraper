package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

// TestBuildLaunchConfig pins how the browser-channel knob (option B: launch
// via a real installed Chrome instead of go-rod's auto-resolved browser) is
// wired, without requiring a real browser to be installed. lookPath is
// injected so the "chrome" case doesn't depend on what's actually installed
// on the machine running the test.
func TestBuildLaunchConfig(t *testing.T) {
	t.Run("default channel has no explicit binary requirement", func(t *testing.T) {
		cfg, err := buildLaunchConfig("", launcher.LookPath)
		if err != nil {
			t.Fatalf("buildLaunchConfig(\"\") error = %v, want nil", err)
		}
		if cfg.binPath != "" {
			t.Errorf("binPath = %q, want empty (let go-rod auto-resolve)", cfg.binPath)
		}
	})

	t.Run("chrome channel resolves a real installed Chrome", func(t *testing.T) {
		lookPath := func() (string, bool) { return "/usr/bin/google-chrome", true }
		cfg, err := buildLaunchConfig("chrome", lookPath)
		if err != nil {
			t.Fatalf("buildLaunchConfig(\"chrome\") error = %v, want nil", err)
		}
		if cfg.binPath != "/usr/bin/google-chrome" {
			t.Errorf("binPath = %q, want %q", cfg.binPath, "/usr/bin/google-chrome")
		}
	})

	t.Run("chrome channel errors when no real Chrome is found", func(t *testing.T) {
		lookPath := func() (string, bool) { return "", false }
		_, err := buildLaunchConfig("chrome", lookPath)
		if err == nil {
			t.Fatal("buildLaunchConfig(\"chrome\") error = nil, want an error when no Chrome is found")
		}
	})

	t.Run("stealth flags are applied regardless of channel", func(t *testing.T) {
		for _, channel := range []string{"", "chrome"} {
			lookPath := func() (string, bool) { return "/usr/bin/google-chrome", true }
			cfg, err := buildLaunchConfig(channel, lookPath)
			if err != nil {
				t.Fatalf("channel %q: buildLaunchConfig error = %v", channel, err)
			}
			found := false
			for _, v := range cfg.flags["disable-blink-features"] {
				if v == "AutomationControlled" {
					found = true
				}
			}
			if !found {
				t.Errorf("channel %q: stealth flags missing disable-blink-features=AutomationControlled, got %v", channel, cfg.flags)
			}
		}
	})
}

func TestBuildURL(t *testing.T) {
	u := buildURL(searchBase, models.SearchParams{
		Keyword: "golang 後端",
		Page:    2,
		Days:    30,
		Order:   2,
		Asc:     0,
	})
	for _, want := range []string{"page=2", "isnew=30", "order=2", "asc=0"} {
		if !strings.Contains(u, want) {
			t.Errorf("URL %q missing %q", u, want)
		}
	}
	if !strings.HasPrefix(u, searchBase) {
		t.Errorf("URL %q does not start with %q", u, searchBase)
	}
}

// TestIsCloudflareChallenge pins the classifier against a real Cloudflare
// managed-challenge response captured from www.104.com.tw (2026-07-04) so the
// detector is validated against production behavior, not a guess.
func TestIsCloudflareChallenge(t *testing.T) {
	challengeBody, err := os.ReadFile("testdata/cloudflare_challenge.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	successBody, err := os.ReadFile("testdata/search_success.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	tests := []struct {
		name    string
		status  int
		headers map[string]string
		body    []byte
		want    bool
	}{
		{
			name:    "real Cloudflare managed challenge",
			status:  http.StatusForbidden,
			headers: map[string]string{"cf-mitigated": "challenge", "content-type": "text/html; charset=UTF-8"},
			body:    challengeBody,
			want:    true,
		},
		{
			name:    "ordinary 403 with no Cloudflare markers",
			status:  http.StatusForbidden,
			headers: map[string]string{"content-type": "text/plain"},
			body:    []byte("access denied"),
			want:    false,
		},
		{
			name:    "successful JSON search response",
			status:  http.StatusOK,
			headers: map[string]string{"content-type": "application/json"},
			body:    successBody,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCloudflareChallenge(tt.status, tt.headers, tt.body)
			if got != tt.want {
				t.Errorf("isCloudflareChallenge(%d, %v, ...) = %v, want %v", tt.status, tt.headers, got, tt.want)
			}
		})
	}
}

// TestSearch_DetectsCloudflareChallenge replays the captured challenge fixture
// through a local server and a real headless browser, so Search's fail-fast
// path is proven end-to-end without ever touching production 104.com.tw.
func TestSearch_DetectsCloudflareChallenge(t *testing.T) {
	challengeBody, err := os.ReadFile("testdata/cloudflare_challenge.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cf-Mitigated", "challenge")
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write(challengeBody)
	}))
	defer srv.Close()

	c, err := New()
	if err != nil {
		t.Skip("skipping: browser init requires network:", err)
	}
	defer c.Close()
	c.baseURL = srv.URL + "/"

	_, err = c.Search(models.SearchParams{Keyword: "golang", Page: 1})
	if !errors.Is(err, ErrCloudflareChallenge) {
		t.Fatalf("Search() error = %v, want errors.Is(err, ErrCloudflareChallenge)", err)
	}
}

// TestSearch_IgnoresUnrelatedCloudflare403 reproduces a real 2026-07-04 CI
// failure: the search page's own JS fires an unrelated ajax call (a
// keyword-suggest autocomplete widget) that Cloudflare 403'd, while the
// actual search API call succeeded moments later. Search must not treat an
// unrelated subresource's block as a block of the whole search.
func TestSearch_IgnoresUnrelatedCloudflare403(t *testing.T) {
	challengeBody, err := os.ReadFile("testdata/cloudflare_challenge.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	successBody, err := os.ReadFile("testdata/search_success.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/search/api/jobs"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(successBody)
		case strings.Contains(r.URL.Path, "/ajax/KeywordSuggest"):
			w.Header().Set("Cf-Mitigated", "challenge")
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write(challengeBody)
		default:
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			fmt.Fprint(w, `<html><body><script>
				fetch("ajax/KeywordSuggest");
				fetch("api/jobs");
			</script></body></html>`)
		}
	}))
	defer srv.Close()

	c, err := New()
	if err != nil {
		t.Skip("skipping: browser init requires network:", err)
	}
	defer c.Close()
	c.baseURL = srv.URL + "/jobs/search/"

	resp, err := c.Search(models.SearchParams{Keyword: "golang", Page: 1})
	if err != nil {
		t.Fatalf("Search() unexpected error: %v, want success despite unrelated 403", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("Search() returned no jobs, want at least 1 from fixture")
	}
}

// TestSearch_ReturnsJobsOnSuccess mirrors the challenge test but replays a
// page whose own JS fetches the search API and gets a normal JSON response,
// proving Search still works when nothing blocks it.
func TestSearch_ReturnsJobsOnSuccess(t *testing.T) {
	successBody, err := os.ReadFile("testdata/search_success.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/search/api/jobs") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(successBody)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		fmt.Fprint(w, `<html><body><script>fetch("api/jobs")</script></body></html>`)
	}))
	defer srv.Close()

	c, err := New()
	if err != nil {
		t.Skip("skipping: browser init requires network:", err)
	}
	defer c.Close()
	c.baseURL = srv.URL + "/jobs/search/"

	resp, err := c.Search(models.SearchParams{Keyword: "golang", Page: 1})
	if err != nil {
		t.Fatalf("Search() unexpected error: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("Search() returned no jobs, want at least 1 from fixture")
	}
}
