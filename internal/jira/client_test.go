package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/longkey1/gojira/internal/config"
)

// capturedRequest records the parts of an HTTP request the tests assert on.
type capturedRequest struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   []byte
}

// newTestServer starts an httptest.Server that records each request and
// responds with the given status and body. It returns a Client pointed at the
// server and a pointer to the last captured request.
func newTestServer(t *testing.T, status int, responseBody string) (*Client, *capturedRequest) {
	t.Helper()

	captured := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		*captured = capturedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.Query(),
			Header: r.Header.Clone(),
			Body:   body,
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(&config.Config{
		BaseURL:  srv.URL,
		Email:    "user@example.com",
		APIToken: "token123",
	})
	return client, captured
}

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		BaseURL:  "https://example.atlassian.net",
		Email:    "user@example.com",
		APIToken: "token123",
	}

	c := NewClient(cfg)

	if c.baseURL != cfg.BaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, cfg.BaseURL)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:token123"))
	if c.authHeader != wantAuth {
		t.Errorf("authHeader = %q, want %q", c.authHeader, wantAuth)
	}
	if c.httpClient == nil {
		t.Error("httpClient = nil, want configured client")
	}
}

func TestRequestHeaders(t *testing.T) {
	client, captured := newTestServer(t, http.StatusOK, `{"id": "1", "key": "PROJ-1"}`)

	if _, err := client.GetIssue(context.Background(), "PROJ-1", nil); err != nil {
		t.Fatalf("GetIssue() unexpected error: %v", err)
	}

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:token123"))
	tests := []struct {
		header string
		want   string
	}{
		{"Authorization", wantAuth},
		{"Content-Type", "application/json"},
		{"Accept", "application/json"},
	}
	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := captured.Header.Get(tt.header); got != tt.want {
				t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestGetIssue(t *testing.T) {
	tests := []struct {
		name        string
		fields      []string
		status      int
		body        string
		wantKey     string
		wantSummary string
		wantQuery   []string // expected values of the "fields" query parameter
		wantErr     string
	}{
		{
			name:        "without fields",
			status:      http.StatusOK,
			body:        `{"id": "10001", "key": "PROJ-123", "fields": {"summary": "hello"}}`,
			wantKey:     "PROJ-123",
			wantSummary: "hello",
		},
		{
			name:        "with fields query parameters",
			fields:      []string{"summary", "status"},
			status:      http.StatusOK,
			body:        `{"id": "10001", "key": "PROJ-123", "fields": {"summary": "hello"}}`,
			wantKey:     "PROJ-123",
			wantSummary: "hello",
			wantQuery:   []string{"summary", "status"},
		},
		{
			name:        "response with invalid escape is sanitized",
			status:      http.StatusOK,
			body:        `{"id": "10001", "key": "PROJ-123", "fields": {"summary": "Mercari\Transaction"}}`,
			wantKey:     "PROJ-123",
			wantSummary: `Mercari\Transaction`,
		},
		{
			name:    "non-200 status returns error with body",
			status:  http.StatusNotFound,
			body:    `{"errorMessages": ["Issue does not exist"]}`,
			wantErr: "request failed with status 404",
		},
		{
			name:    "invalid JSON response returns decode error",
			status:  http.StatusOK,
			body:    `not json`,
			wantErr: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			issue, err := client.GetIssue(context.Background(), "PROJ-123", tt.fields)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("GetIssue() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("GetIssue() error = %q, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetIssue() unexpected error: %v", err)
			}

			if captured.Method != "GET" {
				t.Errorf("method = %q, want GET", captured.Method)
			}
			if captured.Path != "/rest/api/3/issue/PROJ-123" {
				t.Errorf("path = %q, want /rest/api/3/issue/PROJ-123", captured.Path)
			}
			if got := captured.Query["fields"]; !slices.Equal(got, tt.wantQuery) {
				t.Errorf("fields query = %v, want %v", got, tt.wantQuery)
			}
			if issue.Key != tt.wantKey {
				t.Errorf("issue key = %q, want %q", issue.Key, tt.wantKey)
			}
			if issue.Fields.Summary != tt.wantSummary {
				t.Errorf("summary = %q, want %q", issue.Fields.Summary, tt.wantSummary)
			}
		})
	}
}

func TestUpdateIssue(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{
			name:   "success returns nil on 204",
			status: http.StatusNoContent,
		},
		{
			name:    "error status returns error",
			status:  http.StatusBadRequest,
			body:    `{"errorMessages": ["bad field"]}`,
			wantErr: "request failed with status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			reqBody := map[string]any{"fields": map[string]any{"summary": "new summary"}}
			err := client.UpdateIssue(context.Background(), "PROJ-123", reqBody)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("UpdateIssue() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateIssue() unexpected error: %v", err)
			}

			if captured.Method != "PUT" {
				t.Errorf("method = %q, want PUT", captured.Method)
			}
			if captured.Path != "/rest/api/3/issue/PROJ-123" {
				t.Errorf("path = %q, want /rest/api/3/issue/PROJ-123", captured.Path)
			}
			var sent map[string]any
			if err := json.Unmarshal(captured.Body, &sent); err != nil {
				t.Fatalf("request body is not valid JSON: %v", err)
			}
			fields, ok := sent["fields"].(map[string]any)
			if !ok || fields["summary"] != "new summary" {
				t.Errorf("request body = %s, want fields.summary = new summary", captured.Body)
			}
		})
	}
}

func TestCreateIssue(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantKey string
		wantErr string
	}{
		{
			name:    "success on 201",
			status:  http.StatusCreated,
			body:    `{"id": "10001", "key": "PROJ-124"}`,
			wantKey: "PROJ-124",
		},
		{
			name:    "200 is treated as error because 201 is required",
			status:  http.StatusOK,
			body:    `{"id": "10001", "key": "PROJ-124"}`,
			wantErr: "request failed with status 200",
		},
		{
			name:    "error status returns error",
			status:  http.StatusBadRequest,
			body:    `{"errorMessages": ["project is required"]}`,
			wantErr: "request failed with status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			reqBody := map[string]any{"fields": map[string]any{"summary": "created"}}
			issue, err := client.CreateIssue(context.Background(), reqBody)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("CreateIssue() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateIssue() unexpected error: %v", err)
			}

			if captured.Method != "POST" {
				t.Errorf("method = %q, want POST", captured.Method)
			}
			if captured.Path != "/rest/api/3/issue" {
				t.Errorf("path = %q, want /rest/api/3/issue", captured.Path)
			}
			if issue.Key != tt.wantKey {
				t.Errorf("issue key = %q, want %q", issue.Key, tt.wantKey)
			}
		})
	}
}

func TestListComments(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantTotal int
		wantErr   string
	}{
		{
			name:      "success",
			status:    http.StatusOK,
			body:      `{"startAt": 0, "maxResults": 50, "total": 1, "comments": [{"id": "10100"}]}`,
			wantTotal: 1,
		},
		{
			name:    "error status",
			status:  http.StatusForbidden,
			body:    `{}`,
			wantErr: "request failed with status 403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			list, err := client.ListComments(context.Background(), "PROJ-123")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ListComments() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ListComments() unexpected error: %v", err)
			}

			if captured.Method != "GET" || captured.Path != "/rest/api/3/issue/PROJ-123/comment" {
				t.Errorf("request = %s %s, want GET /rest/api/3/issue/PROJ-123/comment",
					captured.Method, captured.Path)
			}
			if list.Total != tt.wantTotal {
				t.Errorf("total = %d, want %d", list.Total, tt.wantTotal)
			}
		})
	}
}

func TestGetComment(t *testing.T) {
	client, captured := newTestServer(t, http.StatusOK, `{"id": "10100"}`)

	comment, err := client.GetComment(context.Background(), "PROJ-123", "10100")
	if err != nil {
		t.Fatalf("GetComment() unexpected error: %v", err)
	}

	if captured.Method != "GET" || captured.Path != "/rest/api/3/issue/PROJ-123/comment/10100" {
		t.Errorf("request = %s %s, want GET /rest/api/3/issue/PROJ-123/comment/10100",
			captured.Method, captured.Path)
	}
	if comment.ID != "10100" {
		t.Errorf("comment ID = %q, want %q", comment.ID, "10100")
	}
}

func TestAddComment(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantID  string
		wantErr string
	}{
		{
			name:   "success on 201",
			status: http.StatusCreated,
			body:   `{"id": "10101"}`,
			wantID: "10101",
		},
		{
			name:    "error status",
			status:  http.StatusBadRequest,
			body:    `{}`,
			wantErr: "request failed with status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			reqBody := map[string]any{"body": map[string]any{"type": "doc"}}
			comment, err := client.AddComment(context.Background(), "PROJ-123", reqBody)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("AddComment() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("AddComment() unexpected error: %v", err)
			}

			if captured.Method != "POST" || captured.Path != "/rest/api/3/issue/PROJ-123/comment" {
				t.Errorf("request = %s %s, want POST /rest/api/3/issue/PROJ-123/comment",
					captured.Method, captured.Path)
			}
			if comment.ID != tt.wantID {
				t.Errorf("comment ID = %q, want %q", comment.ID, tt.wantID)
			}
		})
	}
}

func TestUpdateComment(t *testing.T) {
	client, captured := newTestServer(t, http.StatusOK, `{"id": "10100"}`)

	reqBody := map[string]any{"body": map[string]any{"type": "doc"}}
	comment, err := client.UpdateComment(context.Background(), "PROJ-123", "10100", reqBody)
	if err != nil {
		t.Fatalf("UpdateComment() unexpected error: %v", err)
	}

	if captured.Method != "PUT" || captured.Path != "/rest/api/3/issue/PROJ-123/comment/10100" {
		t.Errorf("request = %s %s, want PUT /rest/api/3/issue/PROJ-123/comment/10100",
			captured.Method, captured.Path)
	}
	if comment.ID != "10100" {
		t.Errorf("comment ID = %q, want %q", comment.ID, "10100")
	}
}

func TestDeleteComment(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr string
	}{
		{
			name:   "success on 204",
			status: http.StatusNoContent,
		},
		{
			name:    "error status",
			status:  http.StatusNotFound,
			wantErr: "request failed with status 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, "")

			err := client.DeleteComment(context.Background(), "PROJ-123", "10100")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("DeleteComment() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DeleteComment() unexpected error: %v", err)
			}

			if captured.Method != "DELETE" || captured.Path != "/rest/api/3/issue/PROJ-123/comment/10100" {
				t.Errorf("request = %s %s, want DELETE /rest/api/3/issue/PROJ-123/comment/10100",
					captured.Method, captured.Path)
			}
		})
	}
}

func TestGetFields(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantCount int
		wantErr   string
	}{
		{
			name:   "success",
			status: http.StatusOK,
			body: `[
				{"id": "summary", "name": "Summary", "custom": false},
				{"id": "customfield_10001", "name": "Sprint", "custom": true}
			]`,
			wantCount: 2,
		},
		{
			name:    "error status",
			status:  http.StatusUnauthorized,
			body:    `{}`,
			wantErr: "request failed with status 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			fields, err := client.GetFields(context.Background())
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("GetFields() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetFields() unexpected error: %v", err)
			}

			if captured.Method != "GET" || captured.Path != "/rest/api/3/field" {
				t.Errorf("request = %s %s, want GET /rest/api/3/field", captured.Method, captured.Path)
			}
			if len(fields) != tt.wantCount {
				t.Fatalf("got %d fields, want %d", len(fields), tt.wantCount)
			}
			if fields[0].ID != "summary" || fields[1].Custom != true {
				t.Errorf("fields = %+v, want summary first and custom Sprint second", fields)
			}
		})
	}
}
