package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/yuanyu90221/104-job-scraper/internal/models"
)

// Format controls the output format of job results.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatCSV   Format = "csv"
)

// Print writes jobs to w in the requested format.
func Print(w io.Writer, jobs []models.Job, format Format) error {
	switch format {
	case FormatTable:
		return printTable(w, jobs)
	case FormatJSON:
		return printJSON(w, jobs)
	case FormatCSV:
		return printCSV(w, jobs)
	default:
		return fmt.Errorf("unknown format %q; use table, json, or csv", format)
	}
}

func printTable(w io.Writer, jobs []models.Job) error {
	table := tablewriter.NewTable(w,
		tablewriter.WithHeader([]string{"#", "職缺名稱", "公司", "地點", "薪資", "刊登日期"}),
	)

	for i, j := range jobs {
		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			j.JobName,
			j.CustName,
			j.JobAddrNoDesc,
			salaryDesc(j),
			formatDate(j.AppearDate),
		})
	}

	return table.Render()
}

func printJSON(w io.Writer, jobs []models.Job) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jobs)
}

func printCSV(w io.Writer, jobs []models.Job) error {
	fmt.Fprintln(w, "職缺名稱,公司,地點,薪資,刊登日期")
	for _, j := range jobs {
		fmt.Fprintf(w, "%s,%s,%s,%s,%s\n",
			csvEscape(j.JobName),
			csvEscape(j.CustName),
			csvEscape(j.JobAddrNoDesc),
			csvEscape(salaryDesc(j)),
			formatDate(j.AppearDate),
		)
	}
	return nil
}

const salaryOpenEnd = 9999999

// salaryDesc builds a human-readable salary string from the job's salary fields.
func salaryDesc(j models.Job) string {
	if j.SalaryLow > 0 && j.SalaryHigh > 0 && j.SalaryHigh < salaryOpenEnd {
		return fmt.Sprintf("%d~%d", j.SalaryLow, j.SalaryHigh)
	}
	if j.SalaryLow > 0 {
		return fmt.Sprintf("%d 以上", j.SalaryLow)
	}
	return "面議"
}

// formatDate converts YYYYMMDD to YYYY-MM-DD for display.
func formatDate(d string) string {
	if len(d) == 8 {
		return d[:4] + "-" + d[4:6] + "-" + d[6:]
	}
	return d
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, `,"`) {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
