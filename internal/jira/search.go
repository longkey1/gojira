package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/longkey1/gojira/internal/models"
)

const (
	defaultMaxResults = 50
	searchEndpoint    = "/rest/api/3/search/jql"
)

type SearchOptions struct {
	Fields     []string
	MaxResults int
}

type searchRequest struct {
	JQL           string   `json:"jql"`
	Fields        []string `json:"fields,omitempty"`
	MaxResults    int      `json:"maxResults,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

type searchResponse struct {
	Issues        []models.Issue `json:"issues"`
	Total         int            `json:"total"`
	IsLast        bool           `json:"isLast"`
	NextPageToken string         `json:"nextPageToken,omitempty"`
}

func (c *Client) SearchJQL(ctx context.Context, jql string, opts SearchOptions, nextPageToken string) (*searchResponse, error) {
	if opts.MaxResults == 0 {
		opts.MaxResults = defaultMaxResults
	}

	reqBody := searchRequest{
		JQL:           jql,
		Fields:        opts.Fields,
		MaxResults:    opts.MaxResults,
		NextPageToken: nextPageToken,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := c.newRequestWithBody("POST", searchEndpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return &result, nil
}

func (c *Client) SearchJQLAll(ctx context.Context, jql string, fields []string) ([]models.Issue, error) {
	var allIssues []models.Issue
	nextPageToken := ""

	for {
		result, err := c.SearchJQL(ctx, jql, SearchOptions{
			Fields:     fields,
			MaxResults: defaultMaxResults,
		}, nextPageToken)
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, result.Issues...)

		if result.IsLast || result.NextPageToken == "" {
			break
		}

		nextPageToken = result.NextPageToken
	}

	return allIssues, nil
}
