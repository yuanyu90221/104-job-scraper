package formatter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

var sampleJobs = []models.Job{
	{
		JobName:         "Golang Engineer",
		JobSalary:       "80000~120000",
		SalaryMonthDesc: "月薪 80,000~120,000",
		PublishDate:     "2026-07-04",
		Company:         models.Company{CompanyName: "Acme Corp"},
		Area:            []models.Area{{AreaDesc: "台北市"}},
	},
}

func TestPrint_Table(t *testing.T) {
	var buf bytes.Buffer
	if err := Print(&buf, sampleJobs, FormatTable); err != nil {
		t.Fatalf("table print error: %v", err)
	}
	if !strings.Contains(buf.String(), "Golang Engineer") {
		t.Error("table output missing job name")
	}
}

func TestPrint_JSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Print(&buf, sampleJobs, FormatJSON); err != nil {
		t.Fatalf("json print error: %v", err)
	}
	if !strings.Contains(buf.String(), `"jobName"`) {
		t.Error("json output missing jobName field")
	}
}

func TestPrint_CSV(t *testing.T) {
	var buf bytes.Buffer
	if err := Print(&buf, sampleJobs, FormatCSV); err != nil {
		t.Fatalf("csv print error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "職缺名稱") {
		t.Error("csv output missing header")
	}
	if !strings.Contains(out, "Golang Engineer") {
		t.Error("csv output missing job name")
	}
}

func TestPrint_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Print(&buf, sampleJobs, "xml")
	if err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}
