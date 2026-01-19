package config

import (
	"errors"
	"os"
)

type Config struct {
	BaseURL  string
	Email    string
	APIToken string
}

func Load() (*Config, error) {
	email := getEnvExpanded("JIRA_EMAIL")
	if email == "" {
		return nil, errors.New("JIRA_EMAIL environment variable is required")
	}

	apiToken := getEnvExpanded("JIRA_API_TOKEN")
	if apiToken == "" {
		return nil, errors.New("JIRA_API_TOKEN environment variable is required")
	}

	baseURL := getEnvExpanded("JIRA_BASE_URL")
	if baseURL == "" {
		return nil, errors.New("JIRA_BASE_URL environment variable is required")
	}

	return &Config{
		BaseURL:  baseURL,
		Email:    email,
		APIToken: apiToken,
	}, nil
}

// getEnvExpanded retrieves an environment variable and expands any ${VAR} or $VAR
// references within its value.
func getEnvExpanded(key string) string {
	value := os.Getenv(key)
	return os.ExpandEnv(value)
}
