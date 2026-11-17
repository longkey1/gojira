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

func TestFindLinkID(t *testing.T) {
	blocks, relates := testLinkTypes[0], testLinkTypes[1]
	// Links as seen from PROJ-1's fields.issuelinks.
	links := []models.IssueLink{
		{ID: "1", Type: blocks, OutwardIssue: &models.Issue{Key: "PROJ-2"}}, // PROJ-1 blocks PROJ-2
		{ID: "2", Type: blocks, InwardIssue: &models.Issue{Key: "PROJ-3"}},  // PROJ-1 is blocked by PROJ-3
		{ID: "3", Type: relates, InwardIssue: &models.Issue{Key: "PROJ-4"}}, // stored as PROJ-4 relates to PROJ-1
		{ID: "4", Type: relates, OutwardIssue: &models.Issue{Key: "PROJ-5"}},
		{ID: "5", Type: relates, InwardIssue: &models.Issue{Key: "PROJ-5"}},
	}

	tests := []struct {
		name     string
		linkType models.IssueLinkType
		outward  string
		wantID   string
		wantErr  string
	}{
		{name: "directional match", linkType: blocks, outward: "PROJ-2", wantID: "1"},
		{name: "issue key is case-insensitive", linkType: blocks, outward: "proj-2", wantID: "1"},
		{name: "directional type does not match reverse direction", linkType: blocks, outward: "PROJ-3", wantErr: "no link found: PROJ-1 blocks PROJ-3"},
		{name: "symmetric type matches reverse direction", linkType: relates, outward: "PROJ-4", wantID: "3"},
		{name: "type must match", linkType: relates, outward: "PROJ-2", wantErr: "no link found"},
		{name: "multiple matches return error with ids", linkType: relates, outward: "PROJ-5", wantErr: "ids: 4, 5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findLinkID(links, tt.linkType, "PROJ-1", tt.outward)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("findLinkID() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("findLinkID() unexpected error: %v", err)
			}
			if got != tt.wantID {
				t.Errorf("findLinkID() = %q, want %q", got, tt.wantID)
			}
		})
	}
}
