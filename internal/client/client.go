package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

// browserChannelEnv selects which browser binary New launches. Empty (the
// default) uses Playwright's bundled Chromium. "chrome" launches a real,
// separately-installed Chrome build (`playwright install chrome`) — spike
// for whether a genuine Chrome fingerprint clears Cloudflare's managed
// challenge more reliably than the bundled, more fingerprintable Chromium.
const browserChannelEnv = "SCRAPER_BROWSER_CHANNEL"

const searchBase = "https://www.104.com.tw/jobs/search/"

// ErrCloudflareChallenge is returned by Search when the response is a
// Cloudflare managed/JS challenge instead of the expected job data, so
// callers can distinguish "blocked" from "no results" or a plain timeout.
var ErrCloudflareChallenge = errors.New("blocked by Cloudflare challenge")

// isCloudflareChallenge reports whether a response looks like Cloudflare's
// managed challenge page rather than real content. Detection is based on
// live evidence captured against www.104.com.tw: a blocked response carries
// a "Cf-Mitigated: challenge" header, and its body is the "Just a moment..."
// interstitial that loads challenges.cloudflare.com.
func isCloudflareChallenge(status int, headers map[string]string, body []byte) bool {
	if status != 403 {
		return false
	}
	if strings.EqualFold(headers["cf-mitigated"], "challenge") {
		return true
	}
	return strings.Contains(string(body), "challenges.cloudflare.com") &&
		strings.Contains(string(body), "Just a moment")
}

// Client drives a headless Chromium browser to bypass Cloudflare protection.
type Client struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext
	baseURL string
}

// chromiumLaunchOptions builds the launch options for the browser, applying
// the same stealth args regardless of channel. An empty channel launches
// Playwright's bundled Chromium; "chrome" launches a real installed Chrome.
func chromiumLaunchOptions(channel string) playwright.BrowserTypeLaunchOptions {
	opts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-blink-features=AutomationControlled",
			"--disable-features=IsolateOrigins,site-per-process",
			"--disable-dev-shm-usage",
			"--no-first-run",
			"--no-default-browser-check",
		},
	}
	if channel != "" {
		opts.Channel = playwright.String(channel)
	}
	return opts
}

// New launches a headless browser with stealth settings to avoid bot
// detection. The browser channel (bundled Chromium vs. a real installed
// Chrome) is controlled by the SCRAPER_BROWSER_CHANNEL env var.
func New() (*Client, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(chromiumLaunchOptions(os.Getenv(browserChannelEnv)))
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("launch chromium: %w", err)
	}

	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.60 Safari/537.36",
		),
		Locale:     playwright.String("zh-TW"),
		TimezoneId: playwright.String("Asia/Taipei"),
		Viewport: &playwright.Size{
			Width:  1280,
			Height: 720,
		},
	})
	if err != nil {
		browser.Close()
		pw.Stop()
		return nil, fmt.Errorf("new context: %w", err)
	}

	if err := ctx.AddInitScript(playwright.Script{
		Content: playwright.String(`
Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3, 4, 5] });
Object.defineProperty(navigator, 'languages', { get: () => ['zh-TW', 'zh', 'en-US', 'en'] });
window.chrome = { runtime: {} };
`),
	}); err != nil {
		ctx.Close()
		browser.Close()
		pw.Stop()
		return nil, fmt.Errorf("add init script: %w", err)
	}

	return &Client{pw: pw, browser: browser, context: ctx, baseURL: searchBase}, nil
}

// Close releases the browser and playwright resources.
func (c *Client) Close() {
	c.context.Close()
	c.browser.Close()
	c.pw.Stop()
}

// Search navigates to the 104 search page and intercepts the JSON API response
// that the page's own JavaScript triggers. A small jitter is applied between
// calls to reduce Cloudflare rate-limiting.
func (c *Client) Search(params models.SearchParams) (*models.SearchResponse, error) {
	if params.Page > 1 {
		time.Sleep(2 * time.Second)
	}
	p, err := c.context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("new page: %w", err)
	}
	defer p.Close()

	respCh := make(chan *models.SearchResponse, 1)
	errCh := make(chan error, 1)

	pageURL := buildURL(c.baseURL, params)

	// isRelevant reports whether a response is either the page navigation
	// itself or the search API call — the only two requests that matter for
	// this search. The page fires other subresource/widget ajax calls (e.g.
	// a keyword-suggest autocomplete) that can independently get a
	// Cloudflare 403 without the actual search being blocked; those must
	// not abort the search.
	isRelevant := func(url string) bool {
		return url == pageURL || strings.Contains(url, "/search/api/jobs")
	}

	p.On("response", func(r playwright.Response) {
		status := r.Status()
		headers := r.Headers()
		url := r.URL()

		if status == 403 {
			if !isRelevant(url) {
				return
			}
			body, _ := r.Body()
			if isCloudflareChallenge(status, headers, body) {
				select {
				case errCh <- fmt.Errorf("%s: %w", url, ErrCloudflareChallenge):
				default:
				}
			}
			return
		}
		if status != 200 {
			return
		}
		ct := headers["content-type"]
		if !strings.Contains(ct, "json") {
			return
		}
		if !strings.Contains(url, "/search/api/jobs") {
			return
		}
		body, err := r.Body()
		if err != nil {
			return
		}
		var sr models.SearchResponse
		if err := json.Unmarshal(body, &sr); err != nil {
			return
		}
		if len(sr.Data) == 0 {
			return
		}
		select {
		case respCh <- &sr:
		default:
		}
	})

	go func() {
		_, _ = p.Goto(pageURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
			Timeout:   playwright.Float(60000),
		})
	}()

	select {
	case result := <-respCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("timeout: no job data captured within 60s (Cloudflare challenge or no results)")
	}
}

func buildURL(baseURL string, params models.SearchParams) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()
	q.Set("jobsource", "joblist_search")
	if params.Keyword != "" {
		q.Set("keyword", params.Keyword)
	}
	if params.Area != "" {
		q.Set("area", params.Area)
	}
	q.Set("page", strconv.Itoa(params.Page))
	if params.Days > 0 {
		q.Set("isnew", strconv.Itoa(params.Days))
	}
	q.Set("order", strconv.Itoa(params.Order))
	q.Set("asc", strconv.Itoa(params.Asc))
	// 104 expects %20 for spaces, not +
	u.RawQuery = strings.ReplaceAll(q.Encode(), "+", "%20")
	return u.String()
}
