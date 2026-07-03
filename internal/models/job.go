package models

// SearchResponse mirrors the 104 API top-level response envelope.
type SearchResponse struct {
	Status    int        `json:"status"`
	StatusMsg string     `json:"statusMsg"`
	Data      SearchData `json:"data"`
}

// SearchData holds pagination metadata and the job list.
type SearchData struct {
	TotalCount int   `json:"totalCount"`
	TotalPage  int   `json:"totalPage"`
	PageNum    int   `json:"pageNum"`
	PageSize   int   `json:"pageSize"`
	List       []Job `json:"list"`
}

// Job represents a single 104 job listing.
type Job struct {
	JobID             string    `json:"jobId"`
	JobName           string    `json:"jobName"`
	JobSalary         string    `json:"jobSalary"`
	SalaryMonthDesc   string    `json:"salaryMonthDesc"`
	SalaryNegotiable  bool      `json:"salaryNegotiable"`
	PublishDate       string    `json:"publishDate"`
	WorkExp           string    `json:"workExp"`
	Edu               string    `json:"edu"`
	Company           Company   `json:"company"`
	Area              []Area    `json:"area"`
}

// Company holds employer information.
type Company struct {
	CompanyID    string `json:"companyId"`
	CompanyName  string `json:"companyName"`
	IndustryDesc string `json:"industryDesc"`
}

// Area represents a work location.
type Area struct {
	AreaCode string `json:"areaCode"`
	AreaDesc string `json:"areaDesc"`
}

// SearchParams holds all query parameters for the 104 search API.
type SearchParams struct {
	Keyword       string
	Area          string
	Page          int
	Days          int    // maps to isnew parameter
	Order         int    // 1=relevance, 2=date, 13=salary
	Asc           int    // 0=desc, 1=asc
	ExpansionType string
}
