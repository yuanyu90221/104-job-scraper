package client

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

// browserChannelEnv selects which browser binary New launches. Empty (the
// default) lets go-rod auto-resolve a browser (system lookup, then
// auto-download if none found). "chrome" requires a real, separately
// installed Chrome — spike for whether a genuine Chrome fingerprint clears
// Cloudflare's managed challenge more reliably than an auto-resolved browser.
const browserChannelEnv = "SCRAPER_BROWSER_CHANNEL"

const searchBase = "https://www.104.com.tw/jobs/search/"

// userAgent, locale, timezone and viewport mirror the previous
// playwright-go BrowserContext defaults; go-rod has no persistent context to
// set these on once, so they're (re)applied to every page New/Search opens.
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.60 Safari/537.36"
const acceptLanguage = "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7"
const timezone = "Asia/Taipei"
const viewportWidth, viewportHeight = 1280, 720

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

// launchConfig is the plain-data result of resolving a browser channel into
// launcher flags/binary, kept free of any *launcher.Launcher so it can be
// unit-tested without launching a real browser.
type launchConfig struct {
	flags   map[string][]string
	binPath string
}

// buildLaunchConfig resolves channel into a launchConfig. lookPath is
// injected (normally launcher.LookPath) so tests can simulate "Chrome
// found"/"Chrome missing" without depending on the machine's actual state.
func buildLaunchConfig(channel string, lookPath func() (string, bool)) (launchConfig, error) {
	cfg := launchConfig{
		flags: map[string][]string{
			"no-sandbox":               nil,
			"disable-setuid-sandbox":   nil,
			"disable-blink-features":   {"AutomationControlled"},
			"disable-features":         {"IsolateOrigins,site-per-process"},
			"disable-dev-shm-usage":    nil,
			"no-first-run":             nil,
			"no-default-browser-check": nil,
		},
	}
	if channel == "chrome" {
		bin, ok := lookPath()
		if !ok {
			return launchConfig{}, fmt.Errorf("%s=chrome requested but no installed Chrome was found", browserChannelEnv)
		}
		cfg.binPath = bin
	}
	return cfg, nil
}

// newLauncher turns a launchConfig into a real *launcher.Launcher.
func newLauncher(cfg launchConfig) *launcher.Launcher {
	l := launcher.New().Headless(true)
	for name, values := range cfg.flags {
		l = l.Set(flags.Flag(name), values...)
	}
	if cfg.binPath != "" {
		l = l.Bin(cfg.binPath)
	}
	return l
}

// Client drives a headless, stealth-patched browser to bypass Cloudflare
// protection.
type Client struct {
	launcher *launcher.Launcher
	browser  *rod.Browser
	baseURL  string
}

// New launches a headless browser with stealth settings to avoid bot
// detection. The browser channel (auto-resolved vs. a real installed
// Chrome) is controlled by the SCRAPER_BROWSER_CHANNEL env var.
func New() (*Client, error) {
	cfg, err := buildLaunchConfig(os.Getenv(browserChannelEnv), launcher.LookPath)
	if err != nil {
		return nil, err
	}

	l := newLauncher(cfg)
	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		l.Cleanup()
		return nil, fmt.Errorf("connect browser: %w", err)
	}

	return &Client{launcher: l, browser: browser, baseURL: searchBase}, nil
}

// Close releases the browser and launcher resources.
func (c *Client) Close() {
	_ = c.browser.Close()
	c.launcher.Cleanup()
}

// newStealthPage opens a fresh stealth-patched page and applies the same
// user-agent/locale/timezone/viewport settings that used to be set once per
// playwright-go BrowserContext.
func newStealthPage(browser *rod.Browser) (*rod.Page, error) {
	page, err := stealth.Page(browser)
	if err != nil {
		return nil, fmt.Errorf("new stealth page: %w", err)
	}

	page.EnableDomain(&proto.NetworkEnable{})

	if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent:      userAgent,
		AcceptLanguage: acceptLanguage,
	}); err != nil {
		_ = page.Close()
		return nil, fmt.Errorf("set user agent: %w", err)
	}

	if err := (proto.EmulationSetTimezoneOverride{TimezoneID: timezone}).Call(page); err != nil {
		_ = page.Close()
		return nil, fmt.Errorf("set timezone: %w", err)
	}

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  viewportWidth,
		Height: viewportHeight,
	}); err != nil {
		_ = page.Close()
		return nil, fmt.Errorf("set viewport: %w", err)
	}

	return page, nil
}

// networkHeadersToMap converts go-rod's CDP header representation into the
// plain map[string]string that isCloudflareChallenge expects.
func networkHeadersToMap(h proto.NetworkHeaders) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[strings.ToLower(k)] = v.String()
	}
	return out
}

// responseBody fetches a response's body via CDP, returning ("", err) if it
// can't be read (e.g. already-consumed/cached responses) rather than
// failing the whole search.
func responseBody(page *rod.Page, requestID proto.NetworkRequestID) ([]byte, error) {
	result, err := (proto.NetworkGetResponseBody{RequestID: requestID}).Call(page)
	if err != nil {
		return nil, err
	}
	if result.Base64Encoded {
		return base64.StdEncoding.DecodeString(result.Body)
	}
	return []byte(result.Body), nil
}

// Search navigates to the 104 search page and intercepts the JSON API response
// that the page's own JavaScript triggers. A small jitter is applied between
// calls to reduce Cloudflare rate-limiting.
func (c *Client) Search(params models.SearchParams) (*models.SearchResponse, error) {
	if params.Page > 1 {
		time.Sleep(2 * time.Second)
	}
	p, err := newStealthPage(c.browser)
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

	// pending tracks relevant responses by request ID between
	// NetworkResponseReceived (headers only) and NetworkLoadingFinished
	// (body fully buffered). Network.getResponseBody fails with "No data
	// found for resource with given identifier" if called before loading
	// finishes, so the body fetch must wait for that second event. Both
	// event types are dispatched on the same single-threaded EachEvent
	// loop, so this map needs no locking.
	type pendingResponse struct {
		status  int
		headers map[string]string
		url     string
	}
	pending := map[proto.NetworkRequestID]pendingResponse{}

	stop := p.EachEvent(
		func(e *proto.NetworkResponseReceived) bool {
			status := e.Response.Status
			headers := networkHeadersToMap(e.Response.Headers)
			url := e.Response.URL

			if status == 403 {
				if !isRelevant(url) {
					return false
				}
			} else if status != 200 || !strings.Contains(headers["content-type"], "json") || !strings.Contains(url, "/search/api/jobs") {
				return false
			}
			pending[e.RequestID] = pendingResponse{status: status, headers: headers, url: url}
			return false
		},
		func(e *proto.NetworkLoadingFinished) bool {
			pr, ok := pending[e.RequestID]
			if !ok {
				return false
			}
			delete(pending, e.RequestID)
			requestID := e.RequestID
			// Body is fetched off the event-dispatch goroutine: a blocking
			// CDP call made synchronously inside an EachEvent callback can
			// deadlock the dispatcher if further events arrive before the
			// call's response does (the dispatcher can't drain the next
			// event until this callback returns, but the connection's
			// reader can't deliver our call's response until the
			// dispatcher drains the events queued ahead of it).
			go func() {
				body, err := responseBody(p, requestID)
				if err != nil {
					return
				}
				if pr.status == 403 {
					if isCloudflareChallenge(pr.status, pr.headers, body) {
						select {
						case errCh <- fmt.Errorf("%s: %w", pr.url, ErrCloudflareChallenge):
						default:
						}
					}
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
			}()
			return false
		},
	)
	go stop()

	go func() {
		_ = p.Navigate(pageURL)
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
