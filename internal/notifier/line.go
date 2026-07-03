package notifier

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

const lineNotifyURL = "https://notify-api.line.me/api/notify"

// LineNotifier sends job summaries to LINE via LINE Notify.
type LineNotifier struct {
	token  string
	client *http.Client
}

// NewLine creates a LineNotifier with the given LINE Notify token.
func NewLine(token string) *LineNotifier {
	return &LineNotifier{
		token:  token,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send formats up to topN jobs into a LINE Notify message and posts it.
func (n *LineNotifier) Send(jobs []models.Job, keyword string, topN int) error {
	if topN <= 0 || topN > len(jobs) {
		topN = len(jobs)
	}

	msg := buildMessage(jobs[:topN], keyword)
	return n.post(msg)
}

func buildMessage(jobs []models.Job, keyword string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "\n📋 104 職缺每日彙報\n")
	fmt.Fprintf(&sb, "🔍 關鍵字：%s\n", keyword)
	fmt.Fprintf(&sb, "📅 日期：%s\n", time.Now().Format("2006-01-02"))
	fmt.Fprintf(&sb, "共找到 %d 筆職缺\n", len(jobs))
	sb.WriteString("─────────────────\n")

	for i, j := range jobs {
		salary := salaryDesc(j)
		date := formatDate(j.AppearDate)

		fmt.Fprintf(&sb, "\n%d. %s\n", i+1, j.JobName)
		fmt.Fprintf(&sb, "   🏢 %s\n", j.CustName)
		fmt.Fprintf(&sb, "   📍 %s\n", j.JobAddrNoDesc)
		fmt.Fprintf(&sb, "   💰 %s\n", salary)
		fmt.Fprintf(&sb, "   📆 %s\n", date)
		fmt.Fprintf(&sb, "   🔗 %s\n", j.Link.Job)
	}

	return sb.String()
}

const salaryOpenEnd = 9999999

func salaryDesc(j models.Job) string {
	if j.SalaryLow > 0 && j.SalaryHigh > 0 && j.SalaryHigh < salaryOpenEnd {
		return fmt.Sprintf("%d~%d", j.SalaryLow, j.SalaryHigh)
	}
	if j.SalaryLow > 0 {
		return fmt.Sprintf("%d 以上", j.SalaryLow)
	}
	return "面議"
}

func formatDate(d string) string {
	if len(d) == 8 {
		return d[:4] + "-" + d[4:6] + "-" + d[6:]
	}
	return d
}

func (n *LineNotifier) post(message string) error {
	const maxLen = 1000
	if len([]rune(message)) > maxLen {
		runes := []rune(message)
		message = string(runes[:maxLen-3]) + "..."
	}

	form := url.Values{}
	form.Set("message", message)

	req, err := http.NewRequest(http.MethodPost, lineNotifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build line notify request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+n.token)

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("line notify post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("line notify returned status %d", resp.StatusCode)
	}

	return nil
}
