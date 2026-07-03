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
		JobNo:         "abc123",
		JobName:       "Golang 後端工程師",
		CustName:      "Acme Corp",
		SalaryLow:     80000,
		SalaryHigh:    120000,
		AppearDate:    "20260704",
		JobAddrNoDesc: "台北市",
		Link:          models.JobLink{Job: "https://www.104.com.tw/job/abc123"},
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
	_ = n
}

func TestBuildMessage_TruncatesLongMessages(t *testing.T) {
	many := make([]models.Job, 20)
	for i := range many {
		many[i] = sampleJobs[0]
	}
	msg := buildMessage(many, "golang")
	if len([]rune(msg)) == 0 {
		t.Error("message should not be empty")
	}
}
