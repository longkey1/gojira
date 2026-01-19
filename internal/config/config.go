package config

import (
	"errors"
	"os"
)

const (
	DefaultBaseURL = "https://your-domain.atlassian.net"
)

type Config struct {
	BaseURL  string
	Email    string
	APIToken string
}

func Load() (*Config, error) {
	email := os.Getenv("JIRA_EMAIL")
	if email == "" {
		return nil, errors.New("JIRA_EMAIL environment variable is required")
	}

	apiToken := os.Getenv("JIRA_API_TOKEN")
	if apiToken == "" {
		return nil, errors.New("JIRA_API_TOKEN environment variable is required")
	}

	baseURL := os.Getenv("JIRA_BASE_URL")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Config{
		BaseURL:  baseURL,
		Email:    email,
		APIToken: apiToken,
	}, nil
}
