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
		area := areaList(j.Area)
		salary := j.SalaryMonthDesc
		if salary == "" {
			salary = j.JobSalary
		}
		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			j.JobName,
			j.Company.CompanyName,
			area,
			salary,
			j.PublishDate,
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
			csvEscape(j.Company.CompanyName),
			csvEscape(areaList(j.Area)),
			csvEscape(j.JobSalary),
			j.PublishDate,
		)
	}
	return nil
}

func areaList(areas []models.Area) string {
	names := make([]string, len(areas))
	for i, a := range areas {
		names[i] = a.AreaDesc
	}
	return strings.Join(names, "/")
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, `,"`) {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
