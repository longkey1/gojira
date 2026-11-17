package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCreateIssueLink(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{name: "success on 201", status: http.StatusCreated},
		{name: "error status returns error", status: http.StatusNotFound, body: `{"errorMessages":["Issue does not exist"]}`, wantErr: "request failed with status 404"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			err := client.CreateIssueLink(context.Background(), "Blocks", "PROJ-1", "PROJ-2")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("CreateIssueLink() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateIssueLink() unexpected error: %v", err)
			}

			if captured.Method != "POST" || captured.Path != "/rest/api/3/issueLink" {
				t.Errorf("request = %s %s, want POST /rest/api/3/issueLink", captured.Method, captured.Path)
			}
			var got map[string]map[string]string
			if err := json.Unmarshal(captured.Body, &got); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if got["type"]["name"] != "Blocks" || got["inwardIssue"]["key"] != "PROJ-1" || got["outwardIssue"]["key"] != "PROJ-2" {
				t.Errorf("request body = %v, want type Blocks, inward PROJ-1, outward PROJ-2", got)
			}
		})
	}
}

func TestListIssueLinkTypes(t *testing.T) {
	client, captured := newTestServer(t, http.StatusOK, `{"issueLinkTypes":[{"id":"10000","name":"Blocks","inward":"is blocked by","outward":"blocks"}]}`)

	types, err := client.ListIssueLinkTypes(context.Background())
	if err != nil {
		t.Fatalf("ListIssueLinkTypes() unexpected error: %v", err)
	}
	if captured.Method != "GET" || captured.Path != "/rest/api/3/issueLinkType" {
		t.Errorf("request = %s %s, want GET /rest/api/3/issueLinkType", captured.Method, captured.Path)
	}
	if len(types) != 1 || types[0].Name != "Blocks" || types[0].Inward != "is blocked by" || types[0].Outward != "blocks" {
		t.Errorf("types = %+v, want one Blocks type", types)
	}
}

func TestGetIssueLinks(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantCount int
		wantErr   string
	}{
		{
			name:   "links are decoded",
			status: http.StatusOK,
			body: `{"key":"PROJ-1","fields":{"issuelinks":[
				{"id":"1","type":{"name":"Blocks","inward":"is blocked by","outward":"blocks"},"outwardIssue":{"key":"PROJ-2","fields":{"summary":"two"}}},
				{"id":"2","type":{"name":"Relates","inward":"relates to","outward":"relates to"},"inwardIssue":{"key":"PROJ-3","fields":{"summary":"three"}}}
			]}}`,
			wantCount: 2,
		},
		{name: "no issuelinks field returns empty slice", status: http.StatusOK, body: `{"key":"PROJ-1","fields":{}}`, wantCount: 0},
		{name: "error status returns error", status: http.StatusNotFound, body: `{}`, wantErr: "request failed with status 404"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			links, err := client.GetIssueLinks(context.Background(), "PROJ-1")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("GetIssueLinks() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetIssueLinks() unexpected error: %v", err)
			}
			if captured.Path != "/rest/api/3/issue/PROJ-1" || captured.Query.Get("fields") != "issuelinks" {
				t.Errorf("request = %s?%s, want /rest/api/3/issue/PROJ-1?fields=issuelinks", captured.Path, captured.Query.Encode())
			}
			if links == nil || len(links) != tt.wantCount {
				t.Fatalf("links = %+v, want %d links", links, tt.wantCount)
			}
			if tt.wantCount == 2 {
				if links[0].OutwardIssue == nil || links[0].OutwardIssue.Key != "PROJ-2" || links[0].OutwardIssue.Fields.Summary != "two" {
					t.Errorf("links[0].OutwardIssue = %+v, want PROJ-2", links[0].OutwardIssue)
				}
				if links[1].InwardIssue == nil || links[1].InwardIssue.Key != "PROJ-3" {
					t.Errorf("links[1].InwardIssue = %+v, want PROJ-3", links[1].InwardIssue)
				}
			}
		})
	}
}

func TestGetIssueLink(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{
			name:   "both sides are decoded",
			status: http.StatusOK,
			body:   `{"id":"10001","type":{"name":"Relates","inward":"relates to","outward":"relates to"},"inwardIssue":{"key":"PROJ-1"},"outwardIssue":{"key":"PROJ-2"}}`,
		},
		{name: "error status returns error", status: http.StatusNotFound, body: `{}`, wantErr: "request failed with status 404"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			link, err := client.GetIssueLink(context.Background(), "10001")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("GetIssueLink() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetIssueLink() unexpected error: %v", err)
			}
			if captured.Method != "GET" || captured.Path != "/rest/api/3/issueLink/10001" {
				t.Errorf("request = %s %s, want GET /rest/api/3/issueLink/10001", captured.Method, captured.Path)
			}
			if link.ID != "10001" || link.Type.Name != "Relates" || link.InwardIssue == nil || link.InwardIssue.Key != "PROJ-1" || link.OutwardIssue == nil || link.OutwardIssue.Key != "PROJ-2" {
				t.Errorf("link = %+v, want Relates PROJ-1 -> PROJ-2", link)
			}
		})
	}
}

func TestDeleteIssueLink(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{name: "success on 204", status: http.StatusNoContent},
		{name: "success on 200", status: http.StatusOK},
		{name: "error status returns error", status: http.StatusNotFound, body: `{"errorMessages":["No issue link with id '1' exists."]}`, wantErr: "request failed with status 404"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newTestServer(t, tt.status, tt.body)

			err := client.DeleteIssueLink(context.Background(), "10001")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("DeleteIssueLink() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DeleteIssueLink() unexpected error: %v", err)
			}
			if captured.Method != "DELETE" || captured.Path != "/rest/api/3/issueLink/10001" {
				t.Errorf("request = %s %s, want DELETE /rest/api/3/issueLink/10001", captured.Method, captured.Path)
			}
		})
	}
}
