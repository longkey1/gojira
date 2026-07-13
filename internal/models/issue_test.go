package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestJiraTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string // raw JSON value
		want     time.Time
		wantZero bool
		wantErr  bool
	}{
		{
			name:  "RFC3339 with colon in timezone",
			input: `"2026-01-16T16:55:41+09:00"`,
			want:  time.Date(2026, 1, 16, 16, 55, 41, 0, time.FixedZone("", 9*60*60)),
		},
		{
			name:  "RFC3339 with milliseconds",
			input: `"2026-01-16T16:55:41.785+09:00"`,
			want:  time.Date(2026, 1, 16, 16, 55, 41, 785_000_000, time.FixedZone("", 9*60*60)),
		},
		{
			name:  "JIRA format without colon in timezone",
			input: `"2026-01-16T16:55:41.785+0900"`,
			want:  time.Date(2026, 1, 16, 16, 55, 41, 785_000_000, time.FixedZone("", 9*60*60)),
		},
		{
			name:  "JIRA format without milliseconds",
			input: `"2026-01-16T16:55:41+0900"`,
			want:  time.Date(2026, 1, 16, 16, 55, 41, 0, time.FixedZone("", 9*60*60)),
		},
		{
			name:  "UTC time",
			input: `"2026-01-16T07:55:41Z"`,
			want:  time.Date(2026, 1, 16, 7, 55, 41, 0, time.UTC),
		},
		{
			name:     "null leaves zero time",
			input:    `null`,
			wantZero: true,
		},
		{
			name:     "empty string leaves zero time",
			input:    `""`,
			wantZero: true,
		},
		{
			name:    "invalid format",
			input:   `"16/01/2026 16:55"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jt JiraTime
			err := jt.UnmarshalJSON([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("UnmarshalJSON(%s) error = nil, want error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("UnmarshalJSON(%s) unexpected error: %v", tt.input, err)
			}
			if tt.wantZero {
				if !jt.IsZero() {
					t.Errorf("UnmarshalJSON(%s) = %v, want zero time", tt.input, jt.Time)
				}
				return
			}
			if !jt.Equal(tt.want) {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v", tt.input, jt.Time, tt.want)
			}
		})
	}
}

func TestJiraTimeMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		time JiraTime
		want string
	}{
		{
			name: "zero time marshals to null",
			time: JiraTime{},
			want: `null`,
		},
		{
			name: "non-zero time marshals to RFC3339",
			time: JiraTime{Time: time.Date(2026, 1, 16, 16, 55, 41, 0, time.UTC)},
			want: `"2026-01-16T16:55:41Z"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.time)
			if err != nil {
				t.Fatalf("Marshal() unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestJiraTimeRoundTrip(t *testing.T) {
	original := `"2026-01-16T16:55:41.785+0900"`

	var jt JiraTime
	if err := jt.UnmarshalJSON([]byte(original)); err != nil {
		t.Fatalf("UnmarshalJSON() unexpected error: %v", err)
	}

	marshaled, err := json.Marshal(jt)
	if err != nil {
		t.Fatalf("Marshal() unexpected error: %v", err)
	}

	// The JIRA offset format is normalized to RFC3339 (and sub-second
	// precision is dropped by time.RFC3339).
	want := `"2026-01-16T16:55:41+09:00"`
	if string(marshaled) != want {
		t.Errorf("round trip = %s, want %s", marshaled, want)
	}
}

func TestFieldsUnmarshalJSON(t *testing.T) {
	t.Run("known fields and custom fields", func(t *testing.T) {
		data := `{
			"summary": "Fix the bug",
			"status": {"id": "1", "name": "In Progress"},
			"issuetype": {"id": "10", "name": "Task", "subtask": false},
			"labels": ["backend", "urgent"],
			"customfield_10001": "sprint-42",
			"customfield_10002": {"value": "High"},
			"unknownfield": "ignored"
		}`

		var f Fields
		if err := json.Unmarshal([]byte(data), &f); err != nil {
			t.Fatalf("Unmarshal() unexpected error: %v", err)
		}

		if f.Summary != "Fix the bug" {
			t.Errorf("Summary = %q, want %q", f.Summary, "Fix the bug")
		}
		if f.Status == nil || f.Status.Name != "In Progress" {
			t.Errorf("Status = %+v, want name %q", f.Status, "In Progress")
		}
		if f.IssueType == nil || f.IssueType.Name != "Task" {
			t.Errorf("IssueType = %+v, want name %q", f.IssueType, "Task")
		}
		if len(f.Labels) != 2 || f.Labels[0] != "backend" {
			t.Errorf("Labels = %v, want [backend urgent]", f.Labels)
		}

		if len(f.CustomFields) != 2 {
			t.Fatalf("CustomFields has %d entries, want 2: %v", len(f.CustomFields), f.CustomFields)
		}
		if got := f.CustomFields["customfield_10001"]; got != "sprint-42" {
			t.Errorf("customfield_10001 = %v, want %q", got, "sprint-42")
		}
		nested, ok := f.CustomFields["customfield_10002"].(map[string]any)
		if !ok || nested["value"] != "High" {
			t.Errorf("customfield_10002 = %v, want map with value High", f.CustomFields["customfield_10002"])
		}
	})

	t.Run("no custom fields yields empty map", func(t *testing.T) {
		var f Fields
		if err := json.Unmarshal([]byte(`{"summary": "s"}`), &f); err != nil {
			t.Fatalf("Unmarshal() unexpected error: %v", err)
		}
		if f.CustomFields == nil {
			t.Fatal("CustomFields = nil, want initialized empty map")
		}
		if len(f.CustomFields) != 0 {
			t.Errorf("CustomFields = %v, want empty", f.CustomFields)
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		var f Fields
		if err := json.Unmarshal([]byte(`{`), &f); err == nil {
			t.Fatal("Unmarshal() error = nil, want error")
		}
	})
}

func TestFieldsMarshalJSON(t *testing.T) {
	t.Run("nil optional fields are omitted", func(t *testing.T) {
		f := Fields{Summary: "Only summary"}

		b, err := json.Marshal(f)
		if err != nil {
			t.Fatalf("Marshal() unexpected error: %v", err)
		}

		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("Unmarshal() unexpected error: %v", err)
		}
		if m["summary"] != "Only summary" {
			t.Errorf("summary = %v, want %q", m["summary"], "Only summary")
		}
		if len(m) != 1 {
			t.Errorf("marshaled map = %v, want only the summary key", m)
		}
	})

	t.Run("custom fields are included", func(t *testing.T) {
		f := Fields{
			Summary: "s",
			Status:  &Status{ID: "1", Name: "Done"},
			CustomFields: map[string]any{
				"customfield_10001": "sprint-42",
			},
		}

		b, err := json.Marshal(f)
		if err != nil {
			t.Fatalf("Marshal() unexpected error: %v", err)
		}

		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("Unmarshal() unexpected error: %v", err)
		}
		if m["customfield_10001"] != "sprint-42" {
			t.Errorf("customfield_10001 = %v, want %q", m["customfield_10001"], "sprint-42")
		}
		status, ok := m["status"].(map[string]any)
		if !ok || status["name"] != "Done" {
			t.Errorf("status = %v, want map with name Done", m["status"])
		}
	})
}

func TestIssueUnmarshalJSON(t *testing.T) {
	data := `{
		"id": "10001",
		"key": "PROJ-123",
		"self": "https://example.atlassian.net/rest/api/3/issue/10001",
		"fields": {
			"summary": "Test issue",
			"created": "2026-01-16T16:55:41.785+0900",
			"assignee": {
				"accountId": "abc123",
				"displayName": "Taro Yamada",
				"active": true
			},
			"parent": {
				"id": "10000",
				"key": "PROJ-100",
				"fields": {"summary": "Parent epic"}
			},
			"customfield_20001": 42
		}
	}`

	var issue Issue
	if err := json.Unmarshal([]byte(data), &issue); err != nil {
		t.Fatalf("Unmarshal() unexpected error: %v", err)
	}

	if issue.Key != "PROJ-123" {
		t.Errorf("Key = %q, want %q", issue.Key, "PROJ-123")
	}
	if issue.ID != "10001" {
		t.Errorf("ID = %q, want %q", issue.ID, "10001")
	}
	if issue.Fields.Summary != "Test issue" {
		t.Errorf("Fields.Summary = %q, want %q", issue.Fields.Summary, "Test issue")
	}
	if issue.Fields.Created == nil {
		t.Fatal("Fields.Created = nil, want parsed time")
	}
	wantCreated := time.Date(2026, 1, 16, 16, 55, 41, 785_000_000, time.FixedZone("", 9*60*60))
	if !issue.Fields.Created.Equal(wantCreated) {
		t.Errorf("Fields.Created = %v, want %v", issue.Fields.Created.Time, wantCreated)
	}
	if issue.Fields.Assignee == nil || issue.Fields.Assignee.DisplayName != "Taro Yamada" {
		t.Errorf("Fields.Assignee = %+v, want display name Taro Yamada", issue.Fields.Assignee)
	}
	if issue.Fields.Parent == nil || issue.Fields.Parent.Fields.Summary != "Parent epic" {
		t.Errorf("Fields.Parent = %+v, want parent summary %q", issue.Fields.Parent, "Parent epic")
	}
	if got := issue.Fields.CustomFields["customfield_20001"]; got != float64(42) {
		t.Errorf("customfield_20001 = %v, want 42", got)
	}
}
