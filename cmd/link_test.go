package cmd

import (
	"strings"
	"testing"

	"github.com/longkey1/gojira/internal/models"
)

var testLinkTypes = []models.IssueLinkType{
	{Name: "Blocks", Inward: "is blocked by", Outward: "blocks"},
	{Name: "Relates", Inward: "relates to", Outward: "relates to"},
}

func TestFindLinkType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantErr  string
	}{
		{name: "exact match", input: "Blocks", wantName: "Blocks"},
		{name: "case-insensitive match returns canonical name", input: "relates", wantName: "Relates"},
		{name: "unknown type lists available names", input: "Foo", wantErr: "available: Blocks, Relates"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findLinkType(testLinkTypes, tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("findLinkType(%q) error = %v, want error containing %q", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("findLinkType(%q) unexpected error: %v", tt.input, err)
			}
			if got.Name != tt.wantName {
				t.Errorf("findLinkType(%q).Name = %q, want %q", tt.input, got.Name, tt.wantName)
			}
		})
	}
}

func TestDescribeLink(t *testing.T) {
	tests := []struct {
		name string
		link models.IssueLink
		want string
	}{
		{
			name: "outward issue uses outward phrase",
			link: models.IssueLink{Type: testLinkTypes[0], OutwardIssue: &models.Issue{Key: "PROJ-2"}},
			want: "PROJ-1 blocks PROJ-2",
		},
		{
			name: "inward issue uses inward phrase",
			link: models.IssueLink{Type: testLinkTypes[0], InwardIssue: &models.Issue{Key: "PROJ-3"}},
			want: "PROJ-1 is blocked by PROJ-3",
		},
		{name: "no linked issue", link: models.IssueLink{Type: testLinkTypes[0]}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describeLink("PROJ-1", tt.link); got != tt.want {
				t.Errorf("describeLink() = %q, want %q", got, tt.want)
			}
		})
	}
}
