package search

import (
	"fmt"

	"github.com/yuanyu90221/104-job-scraper/internal/client"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

// Searcher fetches multiple pages from 104 and aggregates results.
type Searcher struct {
	client *client.Client
}

// New creates a Searcher with a default HTTP client.
func New() *Searcher {
	return &Searcher{client: client.New()}
}

// Run fetches up to maxPages pages for the given params and returns all jobs.
// It stops early if there are no more pages.
func (s *Searcher) Run(params models.SearchParams, maxPages int) ([]models.Job, error) {
	if params.ExpansionType == "" {
		params.ExpansionType = "area,spec,com,job,wf,wktm"
	}

	var jobs []models.Job

	for page := 1; page <= maxPages; page++ {
		params.Page = page

		resp, err := s.client.Search(params)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", page, err)
		}

		jobs = append(jobs, resp.Data.List...)

		if page >= resp.Data.TotalPage {
			break
		}
	}

	return jobs, nil
}
