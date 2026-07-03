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
		salary := j.SalaryMonthDesc
		if salary == "" {
			salary = j.JobSalary
		}
		if salary == "" {
			salary = "薪資面議"
		}

		area := ""
		if len(j.Area) > 0 {
			area = j.Area[0].AreaDesc
		}

		fmt.Fprintf(&sb, "\n%d. %s\n", i+1, j.JobName)
		fmt.Fprintf(&sb, "   🏢 %s\n", j.Company.CompanyName)
		fmt.Fprintf(&sb, "   📍 %s\n", area)
		fmt.Fprintf(&sb, "   💰 %s\n", salary)
		fmt.Fprintf(&sb, "   📆 %s\n", j.PublishDate)
		fmt.Fprintf(&sb, "   🔗 https://www.104.com.tw/job/ajax/content/%s\n", j.JobID)
	}

	return sb.String()
}

func (n *LineNotifier) post(message string) error {
	// LINE Notify 單次訊息上限 1000 字元，超過自動截斷
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
