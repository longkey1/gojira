package jsonutil_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/longkey1/gojira/internal/jsonutil"
	"github.com/longkey1/gojira/internal/models"
)

func TestSanitizeRealJiraFile(t *testing.T) {
	path := "/Users/m-nagai/work/src/github.com/kouzoh/zp-longkey1/reviews/mercari-api-jp/pr21790/context/jira.json"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("Cannot read test file: %v", err)
	}

	t.Logf("Original file size: %d bytes", len(data))

	// Verify original is invalid
	var check any
	origErr := json.Unmarshal(data, &check)
	if origErr != nil {
		t.Logf("Original file is invalid JSON (expected): %v", origErr)
	} else {
		t.Log("Original file is already valid JSON")
	}

	// Sanitize
	sanitized := jsonutil.SanitizeJSON(data)
	t.Logf("Sanitized file size: %d bytes", len(sanitized))

	// Verify sanitized is valid JSON
	var result any
	if err := json.Unmarshal(sanitized, &result); err != nil {
		t.Fatalf("Sanitized JSON is still invalid: %v", err)
	}
	t.Log("Sanitized JSON is valid!")

	// Now test the full pipeline: unmarshal into Issue structs, then marshal back
	// The jira.json file has a wrapper structure, let's check its shape
	arr, ok := result.([]any)
	if ok {
		t.Logf("Top-level array with %d items", len(arr))
		for i, item := range arr {
			m, ok := item.(map[string]any)
			if ok {
				t.Logf("  Item %d keys: %v", i, func() []string {
					keys := make([]string, 0, len(m))
					for k := range m {
						keys = append(keys, k)
					}
					return keys
				}())
			}
		}
	}

	// Re-marshal and verify output is valid JSON
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify the output is valid JSON
	var verify any
	if err := json.Unmarshal(output, &verify); err != nil {
		t.Fatalf("Re-marshaled output is not valid JSON: %v", err)
	}
	t.Log("Re-marshaled output is valid JSON!")

	// Also test with models.Issue struct directly
	// Try to extract the "data" field from each item
	if ok {
		for i, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			dataField, exists := m["data"]
			if !exists {
				continue
			}
			dataJSON, err := json.Marshal(dataField)
			if err != nil {
				t.Errorf("Item %d: Failed to marshal data field: %v", i, err)
				continue
			}

			var issue models.Issue
			if err := json.Unmarshal(dataJSON, &issue); err != nil {
				t.Errorf("Item %d: Failed to unmarshal into Issue: %v", i, err)
				continue
			}

			// Now marshal the issue back
			issueJSON, err := json.MarshalIndent(issue, "", "  ")
			if err != nil {
				t.Errorf("Item %d: Failed to marshal Issue: %v", i, err)
				continue
			}

			var verifyIssue any
			if err := json.Unmarshal(issueJSON, &verifyIssue); err != nil {
				t.Errorf("Item %d: Issue re-marshal is not valid JSON: %v", i, err)
				continue
			}
			t.Logf("Item %d: Issue %s roundtrip OK", i, issue.Key)
		}
	}
}
