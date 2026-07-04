package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

// TestChromiumLaunchOptions pins how the browser-channel spike (option B:
// launch via a real installed Chrome instead of Playwright's bundled
// Chromium) is wired, without requiring a real browser to be installed.
func TestChromiumLaunchOptions(t *testing.T) {
	t.Run("default channel launches bundled Chromium", func(t *testing.T) {
		opts := chromiumLaunchOptions("")
		if opts.Channel != nil {
			t.Errorf("Channel = %v, want nil (bundled Chromium)", *opts.Channel)
		}
	})

	t.Run("chrome channel launches real installed Chrome", func(t *testing.T) {
		opts := chromiumLaunchOptions("chrome")
		if opts.Channel == nil || *opts.Channel != "chrome" {
			t.Errorf("Channel = %v, want %q", opts.Channel, "chrome")
		}
	})

	t.Run("stealth args are applied regardless of channel", func(t *testing.T) {
		for _, channel := range []string{"", "chrome"} {
			opts := chromiumLaunchOptions(channel)
			found := false
			for _, a := range opts.Args {
				if a == "--disable-blink-features=AutomationControlled" {
					found = true
				}
			}
			if !found {
				t.Errorf("channel %q: stealth args missing --disable-blink-features=AutomationControlled", channel)
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
