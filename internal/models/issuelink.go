package models

// IssueLinkType is a link type returned by GET /rest/api/3/issueLinkType.
// Inward and Outward are the phrases shown from each side, e.g. for "Blocks":
// Inward "is blocked by", Outward "blocks".
type IssueLinkType struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Inward  string `json:"inward,omitempty"`
	Outward string `json:"outward,omitempty"`
	Self    string `json:"self,omitempty"`
}

// IssueLinkTypeList is the response body of GET /rest/api/3/issueLinkType.
type IssueLinkTypeList struct {
	IssueLinkTypes []IssueLinkType `json:"issueLinkTypes"`
}

// IssueLink is an entry of an issue's fields.issuelinks.
// Exactly one of InwardIssue / OutwardIssue is set, relative to the issue that
// owns the link: OutwardIssue X means "this issue <Type.Outward> X", and
// InwardIssue X means "this issue <Type.Inward> X".
type IssueLink struct {
	ID           string        `json:"id"`
	Self         string        `json:"self,omitempty"`
	Type         IssueLinkType `json:"type"`
	InwardIssue  *Issue        `json:"inwardIssue,omitempty"`
	OutwardIssue *Issue        `json:"outwardIssue,omitempty"`
}
