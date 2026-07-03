package client

import (
	"strings"
	"testing"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

func TestBuildURL(t *testing.T) {
	u := buildURL(models.SearchParams{
		Keyword: "golang 後端",
		Page:    2,
		Days:    30,
		Order:   2,
		Asc:     0,
	})
	for _, want := range []string{"page=2", "isnew=30", "order=2", "asc=0"} {
		if !strings.Contains(u, want) {
			t.Errorf("URL %q missing %q", u, want)
		}
	}
	if !strings.HasPrefix(u, searchBase) {
		t.Errorf("URL %q does not start with %q", u, searchBase)
	}
}
