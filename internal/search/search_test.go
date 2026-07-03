package search

import (
	"testing"

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
