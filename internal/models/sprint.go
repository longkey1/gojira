package models

// Sprint is a sprint returned by the Jira Agile API.
// Dates are kept as the raw strings the API returns; future sprints have no dates.
type Sprint struct {
	ID            int    `json:"id"`
	Self          string `json:"self,omitempty"`
	State         string `json:"state"`
	Name          string `json:"name"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	CompleteDate  string `json:"completeDate,omitempty"`
	OriginBoardID int    `json:"originBoardId,omitempty"`
	Goal          string `json:"goal,omitempty"`
}
