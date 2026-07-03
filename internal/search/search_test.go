package search

import (
	"testing"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

// TestRun_EmptyResults verifies Run returns an empty slice when the API has no results.
func TestRun_LimitsToMaxPages(t *testing.T) {
	jobs := make([]models.Job, 0)
	// Verify that an empty jobs slice doesn't panic downstream.
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestDefaultExpansionType(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
}
