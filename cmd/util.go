package cmd

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/longkey1/gojira/internal/adf"
)

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
