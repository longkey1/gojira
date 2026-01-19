package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/models"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	authHeader string
}

func NewClient(cfg *config.Config) *Client {
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.Email + ":" + cfg.APIToken))

	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		authHeader: "Basic " + auth,
	}
}

func (c *Client) newRequest(method, endpoint string) (*http.Request, error) {
	return c.newRequestWithBody(method, endpoint, nil)
}

func (c *Client) newRequestWithBody(method, endpoint string, body io.Reader) (*http.Request, error) {
	url := c.baseURL + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Client) GetIssue(ctx context.Context, issueKey string, fields []string) (*models.Issue, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)

	if len(fields) > 0 {
		params := url.Values{}
		for _, f := range fields {
			params.Add("fields", f)
		}
		endpoint += "?" + params.Encode()
	}

	req, err := c.newRequest("GET", endpoint)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var issue models.Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

func (c *Client) GetFields(ctx context.Context) ([]models.Field, error) {
	req, err := c.newRequest("GET", "/rest/api/3/field")
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var fields []models.Field
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return fields, nil
}
