package notifier

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

// maxMessageLen is the LINE Messaging API's text message length limit.
const maxMessageLen = 5000

// LineNotifier sends job summaries to LINE via the Messaging API's push
// message, pushing to a single fixed target (a userId or groupId).
type LineNotifier struct {
	client   *linebot.Client
	targetID string
}

// NewLine creates a LineNotifier that pushes messages to targetID using the
// given Messaging API channel secret and access token. channelSecret is
// required by linebot.New but unused for push messages (it only matters for
// verifying inbound webhook signatures, which this notifier never receives).
func NewLine(channelSecret, channelToken, targetID string, options ...linebot.ClientOption) (*LineNotifier, error) {
	client, err := linebot.New(channelSecret, channelToken, options...)
	if err != nil {
		return nil, fmt.Errorf("new line client: %w", err)
	}
	return &LineNotifier{client: client, targetID: targetID}, nil
}

// Send formats up to topN jobs into a LINE message and pushes it to the
// configured target.
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
	if runes := []rune(message); len(runes) > maxMessageLen {
		message = string(runes[:maxMessageLen-3]) + "..."
	}

	fmt.Fprintf(os.Stderr, "line: pushing message (%d runes)\n", len([]rune(message)))

	if _, err := n.client.PushMessage(n.targetID, linebot.NewTextMessage(message)).Do(); err != nil {
		return fmt.Errorf("line push message: %w", err)
	}

	return nil
}
