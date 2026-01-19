package models

import (
	"strings"
	"time"
)

// JiraTime handles JIRA's date format which may not have colon in timezone
type JiraTime struct {
	time.Time
}

func (jt *JiraTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "null" || s == "" {
		return nil
	}

	// Try standard RFC3339 first
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		jt.Time = t
		return nil
	}

	// Try JIRA format without colon in timezone (e.g., 2026-01-16T16:55:41.785+0900)
	t, err = time.Parse("2006-01-02T15:04:05.000-0700", s)
	if err == nil {
		jt.Time = t
		return nil
	}

	// Try without milliseconds
	t, err = time.Parse("2006-01-02T15:04:05-0700", s)
	if err == nil {
		jt.Time = t
		return nil
	}

	return err
}

func (jt JiraTime) MarshalJSON() ([]byte, error) {
	if jt.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + jt.Format(time.RFC3339) + `"`), nil
}

type Issue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Self   string `json:"self"`
	Fields Fields `json:"fields"`
}

type Fields struct {
	Summary     string      `json:"summary"`
	Description *ADF        `json:"description,omitempty"`
	Status      *Status     `json:"status,omitempty"`
	IssueType   *IssueType  `json:"issuetype,omitempty"`
	Priority    *Priority   `json:"priority,omitempty"`
	Assignee    *User       `json:"assignee,omitempty"`
	Reporter    *User       `json:"reporter,omitempty"`
	Created     *JiraTime   `json:"created,omitempty"`
	Updated     *JiraTime   `json:"updated,omitempty"`
	Labels      []string    `json:"labels,omitempty"`
	Parent      *ParentLink `json:"parent,omitempty"`

	// Custom fields
	StoryPoints *float64 `json:"customfield_12345,omitempty"` // Story Points
	EpicLink    string   `json:"customfield_10006,omitempty"` // Epic Link
}

type ADF struct {
	Type    string `json:"type"`
	Version int    `json:"version,omitempty"`
	Content []any  `json:"content,omitempty"`
}

type Status struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Self           string         `json:"self"`
	StatusCategory StatusCategory `json:"statusCategory,omitempty"`
}

type StatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Self string `json:"self"`
}

type IssueType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Self    string `json:"self"`
	Subtask bool   `json:"subtask"`
}

type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Self string `json:"self"`
}

type User struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress,omitempty"`
	Self         string `json:"self"`
	Active       bool   `json:"active"`
}

type ParentLink struct {
	ID     string       `json:"id"`
	Key    string       `json:"key"`
	Self   string       `json:"self"`
	Fields ParentFields `json:"fields,omitempty"`
}

type ParentFields struct {
	Summary   string     `json:"summary,omitempty"`
	Status    *Status    `json:"status,omitempty"`
	IssueType *IssueType `json:"issuetype,omitempty"`
}
