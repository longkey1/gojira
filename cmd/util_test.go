package cmd

import (
	"slices"
	"testing"
)

func TestExtractIssueKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "raw issue key is returned as-is",
			input: "PROJ-123",
			want:  "PROJ-123",
		},
		{
			name:  "https browse URL",
			input: "https://example.atlassian.net/browse/PROJ-123",
			want:  "PROJ-123",
		},
		{
			name:  "http browse URL",
			input: "http://example.atlassian.net/browse/PROJ-123",
			want:  "PROJ-123",
		},
		{
			name:  "browse URL with query string",
			input: "https://example.atlassian.net/browse/ABC2-9?focusedCommentId=1",
			want:  "ABC2-9",
		},
		{
			name:  "project key with underscore and digits",
			input: "https://example.atlassian.net/browse/A_B1-42",
			want:  "A_B1-42",
		},
		{
			name:    "URL without issue key",
			input:   "https://example.atlassian.net/jira/dashboards",
			wantErr: true,
		},
		{
			name:    "URL with lowercase key does not match",
			input:   "https://example.atlassian.net/browse/proj-123",
			wantErr: true,
		},
		{
			name:  "non-URL string is passed through unvalidated",
			input: "not-a-key",
			want:  "not-a-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractIssueKey(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extractIssueKey(%q) error = nil, want error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractIssueKey(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("extractIssueKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "star all is kept as a single value",
			input: "*all",
			want:  []string{"*all"},
		},
		{
			name:  "star navigable is kept as a single value",
			input: "*navigable",
			want:  []string{"*navigable"},
		},
		{
			name:  "single field",
			input: "summary",
			want:  []string{"summary"},
		},
		{
			name:  "comma separated fields",
			input: "summary,status,assignee",
			want:  []string{"summary", "status", "assignee"},
		},
		{
			name:  "whitespace around fields is trimmed",
			input: " summary , status ",
			want:  []string{"summary", "status"},
		},
		{
			name:  "empty string yields one empty field",
			input: "",
			want:  []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFields(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseFields(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// adfParagraph builds a minimal ADF document containing a single paragraph.
func adfParagraph(text string) map[string]any {
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": text},
				},
			},
		},
	}
}

func TestConvertDescriptionToMarkdown(t *testing.T) {
	t.Run("single issue description becomes markdown string", func(t *testing.T) {
		issue := map[string]any{
			"key": "PROJ-1",
			"fields": map[string]any{
				"summary":     "s",
				"description": adfParagraph("hello world"),
			},
		}

		got, ok := convertDescriptionToMarkdown(issue).(map[string]any)
		if !ok {
			t.Fatalf("convertDescriptionToMarkdown() returned %T, want map[string]any", got)
		}
		fields := got["fields"].(map[string]any)
		if desc, ok := fields["description"].(string); !ok || desc != "hello world" {
			t.Errorf("description = %v, want %q", fields["description"], "hello world")
		}
	})

	t.Run("slice of issues converts each description", func(t *testing.T) {
		issues := []map[string]any{
			{
				"key":    "PROJ-1",
				"fields": map[string]any{"description": adfParagraph("first")},
			},
			{
				"key":    "PROJ-2",
				"fields": map[string]any{"description": adfParagraph("second")},
			},
		}

		got, ok := convertDescriptionToMarkdown(issues).([]map[string]any)
		if !ok {
			t.Fatalf("convertDescriptionToMarkdown() returned %T, want []map[string]any", got)
		}
		want := []string{"first", "second"}
		for i, issue := range got {
			fields := issue["fields"].(map[string]any)
			if desc, ok := fields["description"].(string); !ok || desc != want[i] {
				t.Errorf("issue %d description = %v, want %q", i, fields["description"], want[i])
			}
		}
	})

	t.Run("issue without description is unchanged", func(t *testing.T) {
		issue := map[string]any{
			"key":    "PROJ-1",
			"fields": map[string]any{"summary": "s"},
		}

		got, ok := convertDescriptionToMarkdown(issue).(map[string]any)
		if !ok {
			t.Fatalf("convertDescriptionToMarkdown() returned %T, want map[string]any", got)
		}
		fields := got["fields"].(map[string]any)
		if _, exists := fields["description"]; exists {
			t.Errorf("fields = %v, want no description key", fields)
		}
	})

	t.Run("unmarshalable data is returned unchanged", func(t *testing.T) {
		data := make(chan int)
		if got := convertDescriptionToMarkdown(data); got != any(data) {
			t.Errorf("convertDescriptionToMarkdown() = %v, want original value", got)
		}
	})
}

func TestConvertCommentBodyToMarkdown(t *testing.T) {
	t.Run("single comment body becomes markdown string", func(t *testing.T) {
		comment := map[string]any{
			"id":   "1",
			"body": adfParagraph("a comment"),
		}

		got, ok := convertCommentBodyToMarkdown(comment).(map[string]any)
		if !ok {
			t.Fatalf("convertCommentBodyToMarkdown() returned %T, want map[string]any", got)
		}
		if body, ok := got["body"].(string); !ok || body != "a comment" {
			t.Errorf("body = %v, want %q", got["body"], "a comment")
		}
	})

	t.Run("comment list converts each body", func(t *testing.T) {
		list := map[string]any{
			"total": 2,
			"comments": []any{
				map[string]any{"id": "1", "body": adfParagraph("first")},
				map[string]any{"id": "2", "body": adfParagraph("second")},
			},
		}

		got, ok := convertCommentBodyToMarkdown(list).(map[string]any)
		if !ok {
			t.Fatalf("convertCommentBodyToMarkdown() returned %T, want map[string]any", got)
		}
		comments := got["comments"].([]any)
		want := []string{"first", "second"}
		for i, c := range comments {
			comment := c.(map[string]any)
			if body, ok := comment["body"].(string); !ok || body != want[i] {
				t.Errorf("comment %d body = %v, want %q", i, comment["body"], want[i])
			}
		}
	})

	t.Run("non-ADF body is left unchanged", func(t *testing.T) {
		comment := map[string]any{
			"id":   "1",
			"body": "already a string",
		}

		got, ok := convertCommentBodyToMarkdown(comment).(map[string]any)
		if !ok {
			t.Fatalf("convertCommentBodyToMarkdown() returned %T, want map[string]any", got)
		}
		if body := got["body"]; body != "already a string" {
			t.Errorf("body = %v, want unchanged string", body)
		}
	})
}
