package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCommentUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, c Comment)
	}{
		{
			name: "full comment",
			input: `{
				"id": "10100",
				"self": "https://example.atlassian.net/rest/api/3/issue/10001/comment/10100",
				"body": {
					"type": "doc",
					"version": 1,
					"content": [{"type": "paragraph", "content": [{"type": "text", "text": "hello"}]}]
				},
				"author": {"accountId": "abc", "displayName": "Taro", "active": true},
				"created": "2026-01-16T16:55:41.785+0900",
				"updated": "2026-01-17T10:00:00.000+0900",
				"updateAuthor": {"accountId": "def", "displayName": "Hanako", "active": true}
			}`,
			check: func(t *testing.T, c Comment) {
				if c.ID != "10100" {
					t.Errorf("ID = %q, want %q", c.ID, "10100")
				}
				if c.Body == nil || c.Body.Type != "doc" || c.Body.Version != 1 {
					t.Errorf("Body = %+v, want doc version 1", c.Body)
				}
				if c.Body != nil && len(c.Body.Content) != 1 {
					t.Errorf("Body.Content has %d entries, want 1", len(c.Body.Content))
				}
				if c.Author == nil || c.Author.DisplayName != "Taro" {
					t.Errorf("Author = %+v, want display name Taro", c.Author)
				}
				if c.UpdateAuthor == nil || c.UpdateAuthor.DisplayName != "Hanako" {
					t.Errorf("UpdateAuthor = %+v, want display name Hanako", c.UpdateAuthor)
				}
				wantCreated := time.Date(2026, 1, 16, 16, 55, 41, 785_000_000, time.FixedZone("", 9*60*60))
				if c.Created == nil || !c.Created.Equal(wantCreated) {
					t.Errorf("Created = %v, want %v", c.Created, wantCreated)
				}
			},
		},
		{
			name:  "minimal comment leaves optional fields nil",
			input: `{"id": "1", "self": "https://example.atlassian.net/c/1"}`,
			check: func(t *testing.T, c Comment) {
				if c.ID != "1" {
					t.Errorf("ID = %q, want %q", c.ID, "1")
				}
				if c.Body != nil || c.Author != nil || c.Created != nil || c.Updated != nil || c.UpdateAuthor != nil {
					t.Errorf("optional fields = %+v, want all nil", c)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Comment
			if err := json.Unmarshal([]byte(tt.input), &c); err != nil {
				t.Fatalf("Unmarshal() unexpected error: %v", err)
			}
			tt.check(t, c)
		})
	}
}

func TestCommentListUnmarshalJSON(t *testing.T) {
	input := `{
		"startAt": 0,
		"maxResults": 50,
		"total": 2,
		"comments": [
			{"id": "1", "self": "https://example.atlassian.net/c/1"},
			{"id": "2", "self": "https://example.atlassian.net/c/2"}
		]
	}`

	var list CommentList
	if err := json.Unmarshal([]byte(input), &list); err != nil {
		t.Fatalf("Unmarshal() unexpected error: %v", err)
	}

	if list.StartAt != 0 || list.MaxResults != 50 || list.Total != 2 {
		t.Errorf("pagination = startAt %d, maxResults %d, total %d; want 0, 50, 2",
			list.StartAt, list.MaxResults, list.Total)
	}
	if len(list.Comments) != 2 {
		t.Fatalf("Comments has %d entries, want 2", len(list.Comments))
	}
	if list.Comments[0].ID != "1" || list.Comments[1].ID != "2" {
		t.Errorf("Comments IDs = [%s, %s], want [1, 2]", list.Comments[0].ID, list.Comments[1].ID)
	}
}

func TestCommentMarshalOmitsEmptyOptionalFields(t *testing.T) {
	c := Comment{ID: "1", Self: "https://example.atlassian.net/c/1"}

	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal() unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("Unmarshal() unexpected error: %v", err)
	}
	for _, key := range []string{"body", "author", "created", "updated", "updateAuthor"} {
		if _, ok := m[key]; ok {
			t.Errorf("marshaled JSON contains %q, want it omitted", key)
		}
	}
}
