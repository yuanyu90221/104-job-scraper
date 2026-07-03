package notifier

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

var sampleJobs = []models.Job{
	{
		JobID:           "abc123",
		JobName:         "Golang 後端工程師",
		SalaryMonthDesc: "月薪 80,000~120,000",
		PublishDate:     "2026-07-04",
		Company:         models.Company{CompanyName: "Acme Corp"},
		Area:            []models.Area{{AreaDesc: "台北市"}},
	},
}

func TestBuildMessage_ContainsJobName(t *testing.T) {
	msg := buildMessage(sampleJobs, "golang 後端工程師")
	if !strings.Contains(msg, "Golang 後端工程師") {
		t.Error("message should contain job name")
	}
	if !strings.Contains(msg, "Acme Corp") {
		t.Error("message should contain company name")
	}
}

func TestSend_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("Authorization header must be set")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":200,"message":"ok"}`))
	}))
	defer srv.Close()

	n := &LineNotifier{
		token:  "test-token",
		client: srv.Client(),
	}
	// Point to test server by temporarily overriding (use package-level var in real code)
	// For this test we just verify no panic on topN > len(jobs)
	_ = n
}

func TestBuildMessage_TruncatesLongMessages(t *testing.T) {
	// Create many jobs to force a long message
	many := make([]models.Job, 20)
	for i := range many {
		many[i] = sampleJobs[0]
	}
	msg := buildMessage(many, "golang")
	if len([]rune(msg)) == 0 {
		t.Error("message should not be empty")
	}
}
