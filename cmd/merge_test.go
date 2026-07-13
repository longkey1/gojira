package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/longkey1/gojira/internal/models"
)

func jiraTime(t time.Time) *models.JiraTime {
	return &models.JiraTime{Time: t}
}

func TestIsNewer(t *testing.T) {
	earlier := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	issueWith := func(updated *models.JiraTime) models.Issue {
		return models.Issue{Fields: models.Fields{Updated: updated}}
	}

	tests := []struct {
		name string
		a, b models.Issue
		want bool
	}{
		{
			name: "a updated later than b",
			a:    issueWith(jiraTime(later)),
			b:    issueWith(jiraTime(earlier)),
			want: true,
		},
		{
			name: "a updated earlier than b",
			a:    issueWith(jiraTime(earlier)),
			b:    issueWith(jiraTime(later)),
			want: false,
		},
		{
			name: "equal update times are not newer",
			a:    issueWith(jiraTime(earlier)),
			b:    issueWith(jiraTime(earlier)),
			want: false,
		},
		{
			name: "a without updated is never newer",
			a:    issueWith(nil),
			b:    issueWith(jiraTime(earlier)),
			want: false,
		},
		{
			name: "b without updated makes a newer",
			a:    issueWith(jiraTime(earlier)),
			b:    issueWith(nil),
			want: true,
		},
		{
			name: "both without updated keeps existing",
			a:    issueWith(nil),
			b:    issueWith(nil),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewer(tt.a, tt.b); got != tt.want {
				t.Errorf("isNewer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindJSONFiles(t *testing.T) {
	// Layout:
	//   dir/a.json
	//   dir/b.json
	//   dir/notes.txt
	//   dir/sub/c.json
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}
	for _, name := range []string{
		filepath.Join(dir, "a.json"),
		filepath.Join(dir, "b.json"),
		filepath.Join(dir, "notes.txt"),
		filepath.Join(sub, "c.json"),
	} {
		if err := os.WriteFile(name, []byte("[]"), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	tests := []struct {
		name      string
		pattern   string
		recursive bool
		want      []string
		wantErr   bool
	}{
		{
			name:    "non-recursive matches only top level",
			pattern: "*.json",
			want:    []string{filepath.Join(dir, "a.json"), filepath.Join(dir, "b.json")},
		},
		{
			name:      "recursive includes subdirectories",
			pattern:   "*.json",
			recursive: true,
			want: []string{
				filepath.Join(dir, "a.json"),
				filepath.Join(dir, "b.json"),
				filepath.Join(sub, "c.json"),
			},
		},
		{
			name:    "specific pattern",
			pattern: "a.json",
			want:    []string{filepath.Join(dir, "a.json")},
		},
		{
			name:    "no matches returns empty",
			pattern: "*.yaml",
			want:    nil,
		},
		{
			name:    "invalid pattern returns error",
			pattern: "[",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findJSONFiles(dir, tt.pattern, tt.recursive)
			if tt.wantErr {
				if err == nil {
					t.Fatal("findJSONFiles() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("findJSONFiles() unexpected error: %v", err)
			}
			slices.Sort(got)
			if !slices.Equal(got, tt.want) {
				t.Errorf("findJSONFiles() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("missing directory returns error", func(t *testing.T) {
		if _, err := findJSONFiles(filepath.Join(dir, "missing"), "*.json", false); err == nil {
			t.Fatal("findJSONFiles() error = nil, want error")
		}
	})
}

func TestLoadIssuesFromFile(t *testing.T) {
	writeFile := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "issues.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		return path
	}

	tests := []struct {
		name     string
		content  string
		wantKeys []string
		wantErr  bool
	}{
		{
			name:     "valid issue array",
			content:  `[{"id": "1", "key": "PROJ-1"}, {"id": "2", "key": "PROJ-2"}]`,
			wantKeys: []string{"PROJ-1", "PROJ-2"},
		},
		{
			name:     "empty array",
			content:  `[]`,
			wantKeys: nil,
		},
		{
			name:     "invalid escape sequence is sanitized",
			content:  `[{"id": "1", "key": "PROJ-1", "fields": {"summary": "a\Tb"}}]`,
			wantKeys: []string{"PROJ-1"},
		},
		{
			name:    "single object instead of array",
			content: `{"id": "1", "key": "PROJ-1"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			content: `[`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFile(t, tt.content)

			issues, err := loadIssuesFromFile(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("loadIssuesFromFile() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("loadIssuesFromFile() unexpected error: %v", err)
			}

			var gotKeys []string
			for _, issue := range issues {
				gotKeys = append(gotKeys, issue.Key)
			}
			if !slices.Equal(gotKeys, tt.wantKeys) {
				t.Errorf("issue keys = %v, want %v", gotKeys, tt.wantKeys)
			}
		})
	}

	t.Run("missing file returns error", func(t *testing.T) {
		if _, err := loadIssuesFromFile(filepath.Join(t.TempDir(), "missing.json")); err == nil {
			t.Fatal("loadIssuesFromFile() error = nil, want error")
		}
	})
}
