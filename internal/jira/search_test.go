package jira

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/longkey1/gojira/internal/config"
)

func TestSearchJQL(t *testing.T) {
	tests := []struct {
		name          string
		jql           string
		opts          SearchOptions
		nextPageToken string
		status        int
		responseBody  string
		wantRequest   searchRequest
		wantIssueKeys []string
		wantErr       string
	}{
		{
			name:         "zero MaxResults defaults to 50",
			jql:          "project = PROJ",
			opts:         SearchOptions{},
			status:       http.StatusOK,
			responseBody: `{"issues": [], "isLast": true}`,
			wantRequest: searchRequest{
				JQL:        "project = PROJ",
				MaxResults: defaultMaxResults,
			},
		},
		{
			name: "explicit options are passed through",
			jql:  "assignee = currentUser()",
			opts: SearchOptions{
				Fields:     []string{"summary", "status"},
				MaxResults: 10,
			},
			nextPageToken: "token-abc",
			status:        http.StatusOK,
			responseBody:  `{"issues": [{"id": "1", "key": "PROJ-1"}], "total": 1, "isLast": true}`,
			wantRequest: searchRequest{
				JQL:           "assignee = currentUser()",
				Fields:        []string{"summary", "status"},
				MaxResults:    10,
				NextPageToken: "token-abc",
			},
			wantIssueKeys: []string{"PROJ-1"},
		},
		{
			name:         "response mapping",
			jql:          "project = PROJ",
			opts:         SearchOptions{MaxResults: 2},
			status:       http.StatusOK,
			responseBody: `{"issues": [{"key": "PROJ-1"}, {"key": "PROJ-2"}], "total": 5, "isLast": false, "nextPageToken": "next"}`,
			wantRequest: searchRequest{
				JQL:        "project = PROJ",
				MaxResults: 2,
			},
			wantIssueKeys: []string{"PROJ-1", "PROJ-2"},
		},
		{
			name:         "error status returns error with body",
			jql:          "bad jql (",
			opts:         SearchOptions{},
			status:       http.StatusBadRequest,
			responseBody: `{"errorMessages": ["Error in the JQL Query"]}`,
			wantErr:      "search request failed with status 400",
		},
		{
			name:         "invalid JSON response returns decode error",
			jql:          "project = PROJ",
			opts:         SearchOptions{},
			status:       http.StatusOK,
			responseBody: `not json`,
			wantErr:      "failed to decode search response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer srv.Close()

			client := NewClient(&config.Config{
				BaseURL:  srv.URL,
				Email:    "user@example.com",
				APIToken: "token123",
			})

			result, err := client.SearchJQL(context.Background(), tt.jql, tt.opts, tt.nextPageToken)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("SearchJQL() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("SearchJQL() unexpected error: %v", err)
			}

			if gotMethod != "POST" {
				t.Errorf("method = %q, want POST", gotMethod)
			}
			if gotPath != searchEndpoint {
				t.Errorf("path = %q, want %q", gotPath, searchEndpoint)
			}

			var sent searchRequest
			if err := json.Unmarshal(gotBody, &sent); err != nil {
				t.Fatalf("request body is not valid JSON: %v", err)
			}
			if sent.JQL != tt.wantRequest.JQL {
				t.Errorf("request JQL = %q, want %q", sent.JQL, tt.wantRequest.JQL)
			}
			if sent.MaxResults != tt.wantRequest.MaxResults {
				t.Errorf("request MaxResults = %d, want %d", sent.MaxResults, tt.wantRequest.MaxResults)
			}
			if !slices.Equal(sent.Fields, tt.wantRequest.Fields) {
				t.Errorf("request Fields = %v, want %v", sent.Fields, tt.wantRequest.Fields)
			}
			if sent.NextPageToken != tt.wantRequest.NextPageToken {
				t.Errorf("request NextPageToken = %q, want %q", sent.NextPageToken, tt.wantRequest.NextPageToken)
			}

			var gotKeys []string
			for _, issue := range result.Issues {
				gotKeys = append(gotKeys, issue.Key)
			}
			if !slices.Equal(gotKeys, tt.wantIssueKeys) {
				t.Errorf("issue keys = %v, want %v", gotKeys, tt.wantIssueKeys)
			}
		})
	}
}

func TestSearchJQLAll(t *testing.T) {
	t.Run("paginates until isLast", func(t *testing.T) {
		pages := map[string]string{
			"":      `{"issues": [{"key": "PROJ-1"}, {"key": "PROJ-2"}], "isLast": false, "nextPageToken": "page2"}`,
			"page2": `{"issues": [{"key": "PROJ-3"}], "isLast": true}`,
		}

		var requestCount int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			body, _ := io.ReadAll(r.Body)
			var req searchRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("request body is not valid JSON: %v", err)
			}
			page, ok := pages[req.NextPageToken]
			if !ok {
				t.Errorf("unexpected nextPageToken %q", req.NextPageToken)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(page))
		}))
		defer srv.Close()

		client := NewClient(&config.Config{BaseURL: srv.URL, Email: "e", APIToken: "t"})

		issues, err := client.SearchJQLAll(context.Background(), "project = PROJ", []string{"summary"})
		if err != nil {
			t.Fatalf("SearchJQLAll() unexpected error: %v", err)
		}

		if requestCount != 2 {
			t.Errorf("request count = %d, want 2", requestCount)
		}
		var gotKeys []string
		for _, issue := range issues {
			gotKeys = append(gotKeys, issue.Key)
		}
		want := []string{"PROJ-1", "PROJ-2", "PROJ-3"}
		if !slices.Equal(gotKeys, want) {
			t.Errorf("issue keys = %v, want %v", gotKeys, want)
		}
	})

	t.Run("stops when nextPageToken is empty even if isLast is false", func(t *testing.T) {
		var requestCount int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			_, _ = w.Write([]byte(`{"issues": [{"key": "PROJ-1"}], "isLast": false}`))
		}))
		defer srv.Close()

		client := NewClient(&config.Config{BaseURL: srv.URL, Email: "e", APIToken: "t"})

		issues, err := client.SearchJQLAll(context.Background(), "project = PROJ", nil)
		if err != nil {
			t.Fatalf("SearchJQLAll() unexpected error: %v", err)
		}
		if requestCount != 1 {
			t.Errorf("request count = %d, want 1", requestCount)
		}
		if len(issues) != 1 {
			t.Errorf("got %d issues, want 1", len(issues))
		}
	})

	t.Run("propagates search errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errorMessages": ["boom"]}`))
		}))
		defer srv.Close()

		client := NewClient(&config.Config{BaseURL: srv.URL, Email: "e", APIToken: "t"})

		if _, err := client.SearchJQLAll(context.Background(), "project = PROJ", nil); err == nil {
			t.Fatal("SearchJQLAll() error = nil, want error")
		}
	})
}
