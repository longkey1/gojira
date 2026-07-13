package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFieldUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Field
	}{
		{
			name: "system field",
			input: `{
				"id": "summary",
				"key": "summary",
				"name": "Summary",
				"custom": false,
				"orderable": true,
				"navigable": true,
				"searchable": true,
				"clauseNames": ["summary"],
				"schema": {"type": "string", "system": "summary"}
			}`,
			want: Field{
				ID:          "summary",
				Key:         "summary",
				Name:        "Summary",
				Custom:      false,
				Orderable:   true,
				Navigable:   true,
				Searchable:  true,
				ClauseNames: []string{"summary"},
				Schema:      FieldSchema{Type: "string", System: "summary"},
			},
		},
		{
			name: "custom field with items and custom id",
			input: `{
				"id": "customfield_10001",
				"key": "customfield_10001",
				"name": "Sprint",
				"custom": true,
				"orderable": true,
				"navigable": true,
				"searchable": true,
				"clauseNames": ["cf[10001]", "Sprint"],
				"schema": {
					"type": "array",
					"items": "json",
					"custom": "com.pyxis.greenhopper.jira:gh-sprint",
					"customId": 10001
				}
			}`,
			want: Field{
				ID:          "customfield_10001",
				Key:         "customfield_10001",
				Name:        "Sprint",
				Custom:      true,
				Orderable:   true,
				Navigable:   true,
				Searchable:  true,
				ClauseNames: []string{"cf[10001]", "Sprint"},
				Schema: FieldSchema{
					Type:     "array",
					Items:    "json",
					Custom:   "com.pyxis.greenhopper.jira:gh-sprint",
					CustomID: 10001,
				},
			},
		},
		{
			name:  "missing schema leaves zero value",
			input: `{"id": "issuekey", "name": "Key", "clauseNames": []}`,
			want: Field{
				ID:          "issuekey",
				Name:        "Key",
				ClauseNames: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Field
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("Unmarshal() unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Field = %+v, want %+v", got, tt.want)
			}
		})
	}
}
