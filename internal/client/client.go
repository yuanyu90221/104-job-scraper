package client

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

const (
	baseURL   = "https://www.104.com.tw/jobs/search/list"
	referer   = "https://www.104.com.tw/jobs/search/"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
)

// Client wraps net/http.Client with 104-specific configuration.
type Client struct {
	http    *http.Client
	baseURL string
}

// New creates a Client with a sensible timeout.
func New() *Client {
	return &Client{
		http:    &http.Client{Timeout: 15 * time.Second},
		baseURL: baseURL,
	}
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

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", referer)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var body io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip reader: %w", err)
		}
		defer gr.Close()
		body = gr
	}

	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result models.SearchResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		preview := string(raw)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("decode response: %w\nbody: %s", err, strings.TrimSpace(preview))
	}

	return &result, nil
}
