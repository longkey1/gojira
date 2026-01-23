package cmd

import (
	"encoding/json"
	"os"
	"strings"
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
