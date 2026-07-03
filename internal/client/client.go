package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

const (
	baseURL   = "https://www.104.com.tw/jobs/search/list"
	searchURL = "https://www.104.com.tw/jobs/search/"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
)

// Client wraps net/http.Client with 104-specific configuration.
type Client struct {
	http    *http.Client
	baseURL string
}

// New creates a Client with a cookie jar and visits the search page first
// to collect session cookies, which 104 requires before allowing API access.
func New() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	hc := &http.Client{
		Timeout: 20 * time.Second,
		Jar:     jar,
	}
	c := &Client{http: hc, baseURL: baseURL}

	if err := c.warmup(); err != nil {
		return nil, fmt.Errorf("warmup: %w", err)
	}
	return c, nil
}

// warmup visits the search page to collect session cookies.
func (c *Client) warmup() error {
	req, err := http.NewRequest(http.MethodGet, searchURL, nil)
	if err != nil {
		return err
	}
	setBrowserHeaders(req, "")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// drain body so connection is reused
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// Search fetches one page of job listings from 104.
func (c *Client) Search(params models.SearchParams) (*models.SearchResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	q := u.Query()
	if params.Keyword != "" {
		q.Set("keyword", params.Keyword)
	}
	if params.Area != "" {
		q.Set("area", params.Area)
	}
	q.Set("page", strconv.Itoa(params.Page))
	q.Set("isnew", strconv.Itoa(params.Days))
	q.Set("order", strconv.Itoa(params.Order))
	q.Set("asc", strconv.Itoa(params.Asc))
	q.Set("expansionType", params.ExpansionType)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	setBrowserHeaders(req, searchURL)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result models.SearchResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		preview := string(raw)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("decode response: %w\nbody: %s", err, strings.TrimSpace(preview))
	}

	return &result, nil
}

func setBrowserHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", userAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
}
