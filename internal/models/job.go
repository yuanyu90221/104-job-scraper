package models

// SearchResponse mirrors the 104 /jobs/search/api/jobs response envelope.
type SearchResponse struct {
	Data []Job `json:"data"`
}

// Job represents a single 104 job listing from the /api/jobs endpoint.
type Job struct {
	JobNo          string  `json:"jobNo"`
	JobName        string  `json:"jobName"`
	CustName       string  `json:"custName"`
	CustNo         string  `json:"custNo"`
	CoIndustryDesc string  `json:"coIndustryDesc"`
	SalaryHigh     int     `json:"salaryHigh"`
	SalaryLow      int     `json:"salaryLow"`
	AppearDate     string  `json:"appearDate"` // YYYYMMDD
	JobAddrNoDesc  string  `json:"jobAddrNoDesc"`
	Link           JobLink `json:"link"`
	Description    string  `json:"description"`
}

// JobLink holds URLs for the job listing and company page.
type JobLink struct {
	Job  string `json:"job"`
	Cust string `json:"cust"`
}

// SearchParams holds all query parameters for the 104 search API.
type SearchParams struct {
	Keyword string
	Area    string
	Page    int
	Days    int // maps to isnew parameter
	Order   int // 1=relevance, 2=date, 13=salary
	Asc     int // 0=desc, 1=asc
}

func (j Job) String() string {
	return j.JobNo + " " + j.JobName
}
