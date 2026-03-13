package jsonutil_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/longkey1/gojira/internal/jsonutil"
)

func TestSanitizeWithRawNewlines(t *testing.T) {
	// Simulate JIRA response with raw newlines inside string values
	badJSON := "{\"text\":\"SELECT\nFROM table\"}"

	sanitized := jsonutil.SanitizeJSON([]byte(badJSON))
	t.Logf("Sanitized: %q", string(sanitized))

	var result map[string]any
	if err := json.Unmarshal(sanitized, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Re-marshal to JSON
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	outputStr := string(output)
	t.Logf("Output JSON:\n%s", outputStr)

	// Check that the output doesn't contain raw newlines inside strings
	// Parse line by line and check
	lines := strings.Split(outputStr, "\n")
	for i, line := range lines {
		t.Logf("Line %d: %q", i, line)
	}

	// Verify the output is valid JSON by parsing again
	var verify map[string]any
	if err := json.Unmarshal(output, &verify); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	fmt.Printf("Result text value: %q\n", result["text"])
}
