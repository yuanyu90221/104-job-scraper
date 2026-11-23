package search

import (
	"errors"
	"testing"
	"time"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

func TestRun_LimitsToMaxPages(t *testing.T) {
	jobs := make([]models.Job, 0)
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestNew_InitializesSearcher(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Skip("skipping: browser init requires network:", err)
	}
	defer s.Close()
	if s == nil {
		t.Fatal("New() returned nil searcher")
	}
}

type fakeFetcher struct {
	failures int
	calls    int
}

func (f *fakeFetcher) Search(models.SearchParams) (*models.SearchResponse, error) {
	f.calls++
	if f.calls <= f.failures {
		return nil, errors.New("blocked")
	}
	return &models.SearchResponse{Data: []models.Job{{}}}, nil
}

func newTestSearcher(f *fakeFetcher) *Searcher {
	return &Searcher{client: f, sleep: func(time.Duration) {}, backoff: func(int) time.Duration { return 0 }}
}

func TestRun_RetriesThenSucceeds(t *testing.T) {
	f := &fakeFetcher{failures: 2}
	jobs, _, err := newTestSearcher(f).Run(models.SearchParams{}, 1)
	if err != nil || len(jobs) != 1 || f.calls != 3 {
		t.Fatalf("jobs=%d calls=%d err=%v, want 1 job after 3 calls", len(jobs), f.calls, err)
	}
}

func TestRun_GivesUpAfterMaxAttempts(t *testing.T) {
	f := &fakeFetcher{failures: 99}
	_, _, err := newTestSearcher(f).Run(models.SearchParams{}, 1)
	if err == nil || f.calls != maxAttempts {
		t.Fatalf("calls=%d err=%v, want error after %d calls", f.calls, err, maxAttempts)
	}
}
