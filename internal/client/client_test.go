package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

func TestSearch_Success(t *testing.T) {
	fixture := models.SearchResponse{
		Status:    0,
		StatusMsg: "OK",
		Data: models.SearchData{
			TotalCount: 1,
			TotalPage:  1,
			List: []models.Job{
				{
					JobID:   "abc123",
					JobName: "Golang Engineer",
					Company: models.Company{CompanyName: "Acme Inc."},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") == "" {
			t.Error("Referer header must be set")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fixture)
	}))
	defer srv.Close()

	c := &Client{
		http:    &http.Client{},
		baseURL: srv.URL,
	}

	resp, err := c.Search(models.SearchParams{
		Keyword: "golang",
		Page:    1,
		Days:    30,
		Order:   2,
		Asc:     0,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data.TotalCount != 1 {
		t.Errorf("got TotalCount %d, want 1", resp.Data.TotalCount)
	}
	if resp.Data.List[0].JobName != "Golang Engineer" {
		t.Errorf("got job name %q, want %q", resp.Data.List[0].JobName, "Golang Engineer")
	}
}

func TestSearch_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := &Client{http: &http.Client{}, baseURL: srv.URL}
	_, err := c.Search(models.SearchParams{Page: 1})
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
}
