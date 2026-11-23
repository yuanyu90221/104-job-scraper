package search

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/yuanyu90221/104-job-scraper/internal/client"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

const pageSize = 20

// maxAttempts is how many times a single page is tried (initial + retries).
const maxAttempts = 3

// pageFetcher is the subset of *client.Client that Run needs.
type pageFetcher interface {
	Search(models.SearchParams) (*models.SearchResponse, error)
}

// Searcher fetches multiple pages from 104 and aggregates results.
type Searcher struct {
	client pageFetcher
	closer func()
	// sleep and backoff are injectable so tests don't actually wait.
	sleep   func(time.Duration)
	backoff func(attempt int) time.Duration
}

// defaultBackoff returns a randomized 5-15s delay, growing with attempt.
func defaultBackoff(attempt int) time.Duration {
	base := time.Duration(attempt) * 5 * time.Second
	return base + time.Duration(rand.Int63n(int64(10*time.Second)))
}

// New creates a Searcher backed by a headless browser.
func New() (*Searcher, error) {
	c, err := client.New()
	if err != nil {
		return nil, err
	}
	return &Searcher{client: c, closer: c.Close, sleep: time.Sleep, backoff: defaultBackoff}, nil
}

// Close releases the underlying browser resources.
func (s *Searcher) Close() {
	if s.closer != nil {
		s.closer()
	}
}

// Run fetches up to maxPages pages for the given params and returns all
// jobs, along with the request URL used for page 1 (representative of the
// search — later pages only differ by the page number).
// It stops early when a page returns fewer than pageSize results.
func (s *Searcher) Run(params models.SearchParams, maxPages int) ([]models.Job, string, error) {
	var jobs []models.Job
	var requestURL string

	for page := 1; page <= maxPages; page++ {
		params.Page = page

		resp, err := s.searchWithRetry(params)
		if err != nil {
			return nil, "", fmt.Errorf("page %d: %w", page, err)
		}
		if page == 1 {
			requestURL = resp.RequestURL
		}

		jobs = append(jobs, resp.Data...)

		if len(resp.Data) < pageSize {
			break
		}
	}

	return jobs, requestURL, nil
}

// searchWithRetry retries one page up to maxAttempts times with randomized
// backoff. Each Search call opens a fresh page, so a retry starts from a clean
// page/session (the browser's IP stays the same, which is why a workflow-level
// retry on a new runner exists too).
func (s *Searcher) searchWithRetry(params models.SearchParams) (*models.SearchResponse, error) {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var resp *models.SearchResponse
		resp, err = s.client.Search(params)
		if err == nil {
			return resp, nil
		}
		if attempt == maxAttempts {
			break
		}
		d := s.backoff(attempt)
		fmt.Fprintf(os.Stderr, "[search] page %d attempt %d/%d failed: %v; retrying in %s\n",
			params.Page, attempt, maxAttempts, err, d.Round(time.Second))
		s.sleep(d)
	}
	return nil, fmt.Errorf("after %d attempts: %w", maxAttempts, err)
}
