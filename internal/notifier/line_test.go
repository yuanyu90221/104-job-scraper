package notifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/line/line-bot-sdk-go/v7/linebot"
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

type pushRequest struct {
	method string
	path   string
	auth   string
	body   struct {
		To       string `json:"to"`
		Messages []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"messages"`
	}
}

func newPushServer(t *testing.T, statusCode int, respBody string) (*httptest.Server, *pushRequest) {
	got := &pushRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.path = r.URL.Path
		got.auth = r.Header.Get("Authorization")
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(raw, &got.body); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		w.WriteHeader(statusCode)
		w.Write([]byte(respBody))
	}))
	return srv, got
}

func TestSend_PushesToTargetIDViaMessagingAPI(t *testing.T) {
	srv, got := newPushServer(t, http.StatusOK, `{}`)
	defer srv.Close()

	n, err := NewLine("channel-secret", "channel-token", "U123", linebot.WithEndpointBase(srv.URL))
	if err != nil {
		t.Fatalf("NewLine() error = %v", err)
	}

	if err := n.Send(sampleJobs, "golang 後端工程師", 10); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if got.path != "/v2/bot/message/push" {
		t.Errorf("path = %q, want /v2/bot/message/push", got.path)
	}
	if got.auth != "Bearer channel-token" {
		t.Errorf("Authorization = %q, want %q", got.auth, "Bearer channel-token")
	}
	if got.body.To != "U123" {
		t.Errorf("to = %q, want U123", got.body.To)
	}
	if len(got.body.Messages) != 1 || got.body.Messages[0].Type != "text" {
		t.Fatalf("messages = %+v, want one text message", got.body.Messages)
	}
	if !strings.Contains(got.body.Messages[0].Text, "Golang 後端工程師") {
		t.Errorf("message text = %q, want it to contain the job name", got.body.Messages[0].Text)
	}
}

func TestSend_ReturnsErrorOnAPIFailure(t *testing.T) {
	srv, _ := newPushServer(t, http.StatusBadRequest, `{"message":"bad request"}`)
	defer srv.Close()

	n, err := NewLine("channel-secret", "channel-token", "U123", linebot.WithEndpointBase(srv.URL))
	if err != nil {
		t.Fatalf("NewLine() error = %v", err)
	}

	if err := n.Send(sampleJobs, "golang", 10); err == nil {
		t.Error("Send() error = nil, want non-nil on API failure")
	}
}

func TestNewLine_RequiresChannelSecret(t *testing.T) {
	if _, err := NewLine("", "channel-token", "U123"); err == nil {
		t.Error("NewLine() error = nil, want non-nil when channel secret is empty")
	}
}

func TestNewLine_RequiresChannelToken(t *testing.T) {
	if _, err := NewLine("channel-secret", "", "U123"); err == nil {
		t.Error("NewLine() error = nil, want non-nil when channel token is empty")
	}
}
