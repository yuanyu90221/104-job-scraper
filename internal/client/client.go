package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

const searchBase = "https://www.104.com.tw/jobs/search/"

// Client drives a headless Chromium browser to bypass Cloudflare protection.
type Client struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext
}

// New launches a headless browser with stealth settings to avoid bot detection.
func New() (*Client, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
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
	})
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

	return &Client{pw: pw, browser: browser, context: ctx}, nil
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

	p.On("response", func(r playwright.Response) {
		if r.Status() != 200 {
			return
		}
		ct := r.Headers()["content-type"]
		if !strings.Contains(ct, "json") {
			return
		}
		if !strings.Contains(r.URL(), "/search/api/jobs") {
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

	pageURL := buildURL(params)
	go func() {
		_, _ = p.Goto(pageURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
			Timeout:   playwright.Float(60000),
		})
	}()

	select {
	case result := <-respCh:
		return result, nil
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("timeout: no job data captured within 60s (Cloudflare challenge or no results)")
	}
}

func buildURL(params models.SearchParams) string {
	u, _ := url.Parse(searchBase)
	q := u.Query()
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
