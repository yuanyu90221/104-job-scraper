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

// New launches a headless browser and navigates to the 104 search page once
// to solve the Cloudflare challenge and persist session cookies.
func New() (*Client, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-setuid-sandbox"},
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("launch chromium: %w", err)
	}

	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
		),
	})
	if err != nil {
		browser.Close()
		pw.Stop()
		return nil, fmt.Errorf("new context: %w", err)
	}

	// Warmup: visit 104 to solve Cloudflare challenge and get session cookies.
	p, err := ctx.NewPage()
	if err == nil {
		_, _ = p.Goto(searchBase, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
			Timeout:   playwright.Float(30000),
		})
		p.Close()
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
// that the page's own JavaScript triggers. This bypasses Cloudflare because
// a real browser executes the challenge.
func (c *Client) Search(params models.SearchParams) (*models.SearchResponse, error) {
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
		body, err := r.Body()
		if err != nil {
			return
		}
		var sr models.SearchResponse
		if err := json.Unmarshal(body, &sr); err != nil {
			return
		}
		// Only accept responses that contain real job listing data.
		if sr.Data.TotalPage == 0 && len(sr.Data.List) == 0 {
			return
		}
		select {
		case respCh <- &sr:
		default:
		}
	})

	pageURL := buildURL(params)
	if _, err := p.Goto(pageURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	}); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}

	select {
	case result := <-respCh:
		return result, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("no job data captured (search returned no results or API changed)")
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
	u.RawQuery = q.Encode()
	return u.String()
}
