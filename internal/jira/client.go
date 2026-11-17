package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jsonutil"
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = jsonutil.SanitizeJSON(body)

	var issue models.Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

func (c *Client) UpdateIssue(ctx context.Context, issueKey string, body map[string]any) error {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := c.newRequestWithBody("PUT", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) CreateIssue(ctx context.Context, body map[string]any) (*models.Issue, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := c.newRequestWithBody("POST", "/rest/api/3/issue", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	respBody = jsonutil.SanitizeJSON(respBody)

	var issue models.Issue
	if err := json.Unmarshal(respBody, &issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

func (c *Client) ListComments(ctx context.Context, issueKey string) (*models.CommentList, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)

	req, err := c.newRequest("GET", endpoint)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = jsonutil.SanitizeJSON(body)

	var commentList models.CommentList
	if err := json.Unmarshal(body, &commentList); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &commentList, nil
}

func (c *Client) GetComment(ctx context.Context, issueKey, commentID string) (*models.Comment, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/comment/%s", issueKey, commentID)

	req, err := c.newRequest("GET", endpoint)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = jsonutil.SanitizeJSON(body)

	var comment models.Comment
	if err := json.Unmarshal(body, &comment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &comment, nil
}

func (c *Client) AddComment(ctx context.Context, issueKey string, body map[string]any) (*models.Comment, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := c.newRequestWithBody("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	respBody = jsonutil.SanitizeJSON(respBody)

	var comment models.Comment
	if err := json.Unmarshal(respBody, &comment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &comment, nil
}

func (c *Client) UpdateComment(ctx context.Context, issueKey, commentID string, body map[string]any) (*models.Comment, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/comment/%s", issueKey, commentID)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := c.newRequestWithBody("PUT", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	respBody = jsonutil.SanitizeJSON(respBody)

	var comment models.Comment
	if err := json.Unmarshal(respBody, &comment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &comment, nil
}

func (c *Client) DeleteComment(ctx context.Context, issueKey, commentID string) error {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/comment/%s", issueKey, commentID)

	req, err := c.newRequest("DELETE", endpoint)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = jsonutil.SanitizeJSON(body)

	var fields []models.Field
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return fields, nil
}

// CreateIssueLink creates a link of the given type. Jira reads it as
// "<inwardKey> <type.outward> <outwardKey>", e.g. for "Blocks" the inward issue blocks the outward issue.
func (c *Client) CreateIssueLink(ctx context.Context, typeName, inwardKey, outwardKey string) error {
	jsonBody, err := json.Marshal(map[string]any{
		"type":         map[string]string{"name": typeName},
		"inwardIssue":  map[string]string{"key": inwardKey},
		"outwardIssue": map[string]string{"key": outwardKey},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := c.newRequestWithBody("POST", "/rest/api/3/issueLink", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetIssueLink returns a link with both InwardIssue and OutwardIssue set,
// unlike the entries of fields.issuelinks which only carry the other side.
func (c *Client) GetIssueLink(ctx context.Context, linkID string) (*models.IssueLink, error) {
	req, err := c.newRequest("GET", fmt.Sprintf("/rest/api/3/issueLink/%s", linkID))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = jsonutil.SanitizeJSON(body)

	var link models.IssueLink
	if err := json.Unmarshal(body, &link); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &link, nil
}

func (c *Client) DeleteIssueLink(ctx context.Context, linkID string) error {
	req, err := c.newRequest("DELETE", fmt.Sprintf("/rest/api/3/issueLink/%s", linkID))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) ListIssueLinkTypes(ctx context.Context) ([]models.IssueLinkType, error) {
	req, err := c.newRequest("GET", "/rest/api/3/issueLinkType")
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = jsonutil.SanitizeJSON(body)

	var list models.IssueLinkTypeList
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return list.IssueLinkTypes, nil
}

// GetIssueLinks returns fields.issuelinks of an issue. It does not go through
// GetIssue because models.Fields drops non-custom fields it does not know.
func (c *Client) GetIssueLinks(ctx context.Context, issueKey string) ([]models.IssueLink, error) {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s?fields=issuelinks", issueKey)

	req, err := c.newRequest("GET", endpoint)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = jsonutil.SanitizeJSON(body)

	var issue struct {
		Fields struct {
			IssueLinks []models.IssueLink `json:"issuelinks"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if issue.Fields.IssueLinks == nil {
		return []models.IssueLink{}, nil
	}
	return issue.Fields.IssueLinks, nil
}
