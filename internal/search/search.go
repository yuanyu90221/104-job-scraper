package search

import (
	"fmt"

	"github.com/yuanyu90221/104-job-scraper/internal/client"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

const pageSize = 20

// Searcher fetches multiple pages from 104 and aggregates results.
type Searcher struct {
	client *client.Client
}

// New creates a Searcher backed by a headless browser.
func New() (*Searcher, error) {
	c, err := client.New()
	if err != nil {
		return nil, err
	}
	return &Searcher{client: c}, nil
}

// Close releases the underlying browser resources.
func (s *Searcher) Close() {
	s.client.Close()
}

// Run fetches up to maxPages pages for the given params and returns all jobs.
// It stops early when a page returns fewer than pageSize results.
func (s *Searcher) Run(params models.SearchParams, maxPages int) ([]models.Job, error) {
	var jobs []models.Job

	for page := 1; page <= maxPages; page++ {
		params.Page = page

		resp, err := s.client.Search(params)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", page, err)
		}

		jobs = append(jobs, resp.Data...)

		if len(resp.Data) < pageSize {
			break
		}
	}

	return jobs, nil
}
