package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/longkey1/gojira/internal/models"
	"github.com/spf13/cobra"
)

var (
	mergeDir       string
	mergePattern   string
	mergeRecursive bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge JSON files containing issues",
	Long: `Merge JSON files containing issues from a directory.

Searches for JSON files and merges them.
When duplicate issues are found (same key), the one with the latest updated date is kept.

Examples:
  # Merge all JSON files in a directory
  gojira merge --dir ./output

  # Merge with specific file pattern
  gojira merge --dir ./output --pattern 'issues-*.json'

  # Merge recursively
  gojira merge --dir ./output --recursive`,
	RunE: runMerge,
}

func init() {
	mergeCmd.Flags().StringVar(&mergeDir, "dir", ".", "Directory to search for JSON files")
	mergeCmd.Flags().StringVar(&mergePattern, "pattern", "*.json", "File name pattern (glob)")
	mergeCmd.Flags().BoolVarP(&mergeRecursive, "recursive", "r", false, "Search recursively in subdirectories")
}

func runMerge(cmd *cobra.Command, args []string) error {
	files, err := findJSONFiles(mergeDir, mergePattern, mergeRecursive)
	if err != nil {
		return fmt.Errorf("failed to find JSON files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no JSON files found matching pattern '%s' in '%s'", mergePattern, mergeDir)
	}

	issueMap := make(map[string]models.Issue)

	for _, file := range files {
		issues, err := loadIssuesFromFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", file, err)
			continue
		}

		for _, issue := range issues {
			existing, exists := issueMap[issue.Key]
			if !exists {
				issueMap[issue.Key] = issue
				continue
			}

			if isNewer(issue, existing) {
				issueMap[issue.Key] = issue
			}
		}
	}

	result := make([]models.Issue, 0, len(issueMap))
	for _, issue := range issueMap {
		result = append(result, issue)
	}

	return outputJSON(result)
}

func findJSONFiles(dir, pattern string, recursive bool) ([]string, error) {
	var files []string

	if recursive {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			matched, err := filepath.Match(pattern, info.Name())
			if err != nil {
				return err
			}

			if matched {
				files = append(files, path)
			}

			return nil
		})
		return files, err
	}

	// Non-recursive: only search in the specified directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return nil, err
		}

		if matched {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

func loadIssuesFromFile(path string) ([]models.Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var issues []models.Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}

func isNewer(a, b models.Issue) bool {
	if a.Fields.Updated == nil {
		return false
	}
	if b.Fields.Updated == nil {
		return true
	}
	return a.Fields.Updated.After(b.Fields.Updated.Time)
}
