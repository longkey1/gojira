package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/longkey1/gojira/internal/adf"
)

var issueURLPattern = regexp.MustCompile(`/browse/([A-Z][A-Z0-9_]*-\d+)`)

// extractIssueKey accepts either a raw issue key (e.g. "PROJ-123") or a
// JIRA browse URL (e.g. "https://example.atlassian.net/browse/PROJ-123")
// and returns the issue key.
func extractIssueKey(input string) (string, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		m := issueURLPattern.FindStringSubmatch(input)
		if len(m) < 2 {
			return "", fmt.Errorf("could not extract issue key from URL: %s", input)
		}
		return m[1], nil
	}
	return input, nil
}

func parseFields(fieldsStr string) []string {
	if fieldsStr == "*all" || fieldsStr == "*navigable" {
		return []string{fieldsStr}
	}
	fields := strings.Split(fieldsStr, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}

func outputJSON(data any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// convertDescriptionToMarkdown converts data to a map, then replaces
// fields.description (ADF) with a Markdown string.
// Supports a single issue or a slice of issues.
func convertDescriptionToMarkdown(data any) any {
	b, err := json.Marshal(data)
	if err != nil {
		return data
	}

	// Try as a single issue
	var single map[string]any
	if err := json.Unmarshal(b, &single); err == nil {
		if _, ok := single["fields"]; ok {
			replaceDescription(single)
			return single
		}
	}

	// Try as a slice of issues
	var slice []map[string]any
	if err := json.Unmarshal(b, &slice); err == nil {
		for i := range slice {
			replaceDescription(slice[i])
		}
		return slice
	}

	return data
}

// convertCommentBodyToMarkdown converts comment body (ADF) to Markdown string.
// Supports a single comment or a CommentList.
func convertCommentBodyToMarkdown(data any) any {
	b, err := json.Marshal(data)
	if err != nil {
		return data
	}

	// Try as a CommentList
	var list map[string]any
	if err := json.Unmarshal(b, &list); err == nil {
		if comments, ok := list["comments"].([]any); ok {
			for _, c := range comments {
				if comment, ok := c.(map[string]any); ok {
					replaceCommentBody(comment)
				}
			}
			return list
		}
	}

	// Try as a single comment
	var single map[string]any
	if err := json.Unmarshal(b, &single); err == nil {
		if _, ok := single["body"]; ok {
			replaceCommentBody(single)
			return single
		}
	}

	return data
}

func replaceCommentBody(comment map[string]any) {
	body, ok := comment["body"].(map[string]any)
	if !ok {
		return
	}
	comment["body"] = adf.ToMarkdown(body)
}

func replaceDescription(issue map[string]any) {
	fields, ok := issue["fields"].(map[string]any)
	if !ok {
		return
	}
	desc, ok := fields["description"].(map[string]any)
	if !ok {
		return
	}
	fields["description"] = adf.ToMarkdown(desc)
}
