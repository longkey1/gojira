package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/longkey1/gojira/internal/config"
	"github.com/longkey1/gojira/internal/jira"
	"github.com/longkey1/gojira/internal/models"
	"github.com/spf13/cobra"
)

var (
	linkCreateType    string
	linkCreateInward  string
	linkCreateOutward string

	linkDeleteType    string
	linkDeleteInward  string
	linkDeleteOutward string
)

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Manage issue links",
}

var linkCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a link between two issues",
	Long: `Create a link between two issues.

Jira reads the link as "<inward> <type's outward phrase> <outward>".
For directional types this means the --inward issue is the subject:

  --type Blocks    --inward B --outward A   =>  "B blocks A" / "A is blocked by B"
  --type Duplicate --inward B --outward A   =>  "B duplicates A" / "A is duplicated by B"
  --type Cloners   --inward B --outward A   =>  "B clones A" / "A is cloned by B"

The resulting relation is printed in the "relation" field of the output.
Run 'gojira link types' for the link types and phrases available on your site.

Examples:
  # Relate two issues (non-directional)
  gojira link create --type Relates --inward PROJ-1 --outward PROJ-2

  # "PROJ-2 is blocked by PROJ-1"
  gojira link create --type Blocks --inward PROJ-1 --outward PROJ-2`,
	Annotations: map[string]string{writeAnnotation: "true"},
	Args:        cobra.NoArgs,
	RunE:        runLinkCreate,
}

var linkDeleteCmd = &cobra.Command{
	Use:   "delete [<link-id>]",
	Short: "Delete a link between two issues",
	Long: `Delete an issue link.

Specify the link either by its ID (the "id" in 'gojira link list' output),
or by its type and the two issues with the same meaning as 'gojira link create'.
With flags, the command fails unless exactly one link matches. For types whose
inward and outward phrases are the same (e.g. Relates), the two issues match in
either order.

The deleted relation is printed in the "relation" field of the output.

Examples:
  # Delete by link ID
  gojira link delete 10001

  # Delete "PROJ-1 blocks PROJ-2"
  gojira link delete --type Blocks --inward PROJ-1 --outward PROJ-2`,
	Annotations: map[string]string{writeAnnotation: "true"},
	Args:        cobra.MaximumNArgs(1),
	RunE:        runLinkDelete,
}

var linkListCmd = &cobra.Command{
	Use:   "list <issue-key>",
	Short: "List links of an issue",
	Long: `List the links of an issue (fields.issuelinks).

Each entry has a "relation" field describing the link from the issue's point of view.

Examples:
  gojira link list PROJ-123
  gojira link list https://your-domain.atlassian.net/browse/PROJ-123`,
	Args: cobra.ExactArgs(1),
	RunE: runLinkList,
}

var linkTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List available issue link types",
	Long: `List the issue link types available on the Jira site.

"inward" / "outward" are the phrases shown on each side. With
'gojira link create --inward X --outward Y', Jira reads "X <outward> Y".

Examples:
  gojira link types`,
	Args: cobra.NoArgs,
	RunE: runLinkTypes,
}

func init() {
	linkCmd.AddCommand(linkCreateCmd)
	linkCmd.AddCommand(linkDeleteCmd)
	linkCmd.AddCommand(linkListCmd)
	linkCmd.AddCommand(linkTypesCmd)

	linkCreateCmd.Flags().StringVar(&linkCreateType, "type", "", "Link type name (e.g. Relates, Blocks, Duplicate, Cloners; see 'gojira link types')")
	linkCreateCmd.Flags().StringVar(&linkCreateInward, "inward", "", "Inward issue key or URL (the subject: \"<inward> blocks <outward>\")")
	linkCreateCmd.Flags().StringVar(&linkCreateOutward, "outward", "", "Outward issue key or URL (the object: \"<outward> is blocked by <inward>\")")
	_ = linkCreateCmd.MarkFlagRequired("type")
	_ = linkCreateCmd.MarkFlagRequired("inward")
	_ = linkCreateCmd.MarkFlagRequired("outward")

	linkDeleteCmd.Flags().StringVar(&linkDeleteType, "type", "", "Link type name; use with --inward and --outward instead of <link-id>")
	linkDeleteCmd.Flags().StringVar(&linkDeleteInward, "inward", "", "Inward issue key or URL (the subject: \"<inward> blocks <outward>\")")
	linkDeleteCmd.Flags().StringVar(&linkDeleteOutward, "outward", "", "Outward issue key or URL (the object: \"<outward> is blocked by <inward>\")")
	linkDeleteCmd.MarkFlagsRequiredTogether("type", "inward", "outward")
}

// linkCreateResult is the output of 'link create'. The create API returns no body.
type linkCreateResult struct {
	Type         string `json:"type"`
	InwardIssue  string `json:"inwardIssue"`
	OutwardIssue string `json:"outwardIssue"`
	Relation     string `json:"relation"`
}

// linkDeleteResult is the output of 'link delete'. The delete API returns no body.
type linkDeleteResult struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	InwardIssue  string `json:"inwardIssue"`
	OutwardIssue string `json:"outwardIssue"`
	Relation     string `json:"relation"`
}

// linkListEntry is an issue link with a human-readable relation added.
type linkListEntry struct {
	models.IssueLink
	Relation string `json:"relation"`
}

// findLinkType looks up a link type by name, case-insensitively, so the
// canonical name is sent to Jira and typos fail before the write.
func findLinkType(types []models.IssueLinkType, name string) (models.IssueLinkType, error) {
	for _, t := range types {
		if strings.EqualFold(t.Name, name) {
			return t, nil
		}
	}
	names := make([]string, 0, len(types))
	for _, t := range types {
		names = append(names, t.Name)
	}
	return models.IssueLinkType{}, fmt.Errorf("unknown link type %q (available: %s)", name, strings.Join(names, ", "))
}

// describeLink returns the relation of link from the point of view of issueKey.
func describeLink(issueKey string, link models.IssueLink) string {
	switch {
	case link.OutwardIssue != nil:
		return fmt.Sprintf("%s %s %s", issueKey, link.Type.Outward, link.OutwardIssue.Key)
	case link.InwardIssue != nil:
		return fmt.Sprintf("%s %s %s", issueKey, link.Type.Inward, link.InwardIssue.Key)
	default:
		return ""
	}
}

// findLinkID returns the ID of the single link in inwardKey's links that reads
// "<inwardKey> <linkType.Outward> <outwardKey>". Symmetric types also match the
// reverse direction, since the stored direction is not visible to the user.
func findLinkID(links []models.IssueLink, linkType models.IssueLinkType, inwardKey, outwardKey string) (string, error) {
	symmetric := linkType.Inward == linkType.Outward
	var ids []string
	for _, l := range links {
		if !strings.EqualFold(l.Type.Name, linkType.Name) {
			continue
		}
		if (l.OutwardIssue != nil && strings.EqualFold(l.OutwardIssue.Key, outwardKey)) ||
			(symmetric && l.InwardIssue != nil && strings.EqualFold(l.InwardIssue.Key, outwardKey)) {
			ids = append(ids, l.ID)
		}
	}
	switch len(ids) {
	case 0:
		return "", fmt.Errorf("no link found: %s %s %s", inwardKey, linkType.Outward, outwardKey)
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("multiple links found for %s %s %s (ids: %s); delete by link ID", inwardKey, linkType.Outward, outwardKey, strings.Join(ids, ", "))
	}
}

func runLinkCreate(cmd *cobra.Command, args []string) error {
	inwardKey, err := extractIssueKey(linkCreateInward)
	if err != nil {
		return err
	}
	outwardKey, err := extractIssueKey(linkCreateOutward)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	types, err := client.ListIssueLinkTypes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list issue link types: %w", err)
	}
	linkType, err := findLinkType(types, linkCreateType)
	if err != nil {
		return err
	}

	if err := client.CreateIssueLink(ctx, linkType.Name, inwardKey, outwardKey); err != nil {
		return fmt.Errorf("failed to create issue link: %w", err)
	}

	return outputJSON(linkCreateResult{
		Type:         linkType.Name,
		InwardIssue:  inwardKey,
		OutwardIssue: outwardKey,
		Relation:     fmt.Sprintf("%s %s %s", inwardKey, linkType.Outward, outwardKey),
	})
}

func runLinkDelete(cmd *cobra.Command, args []string) error {
	byFlags := cmd.Flags().Changed("type")
	if len(args) == 1 && byFlags {
		return fmt.Errorf("specify either <link-id> or --type/--inward/--outward, not both")
	}
	if len(args) == 0 && !byFlags {
		return fmt.Errorf("specify <link-id> or --type, --inward and --outward")
	}

	var inwardKey, outwardKey string
	if byFlags {
		var err error
		if inwardKey, err = extractIssueKey(linkDeleteInward); err != nil {
			return err
		}
		if outwardKey, err = extractIssueKey(linkDeleteOutward); err != nil {
			return err
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)
	ctx := context.Background()

	var linkID string
	if byFlags {
		types, err := client.ListIssueLinkTypes(ctx)
		if err != nil {
			return fmt.Errorf("failed to list issue link types: %w", err)
		}
		linkType, err := findLinkType(types, linkDeleteType)
		if err != nil {
			return err
		}
		links, err := client.GetIssueLinks(ctx, inwardKey)
		if err != nil {
			return fmt.Errorf("failed to list issue links: %w", err)
		}
		if linkID, err = findLinkID(links, linkType, inwardKey, outwardKey); err != nil {
			return err
		}
	} else {
		linkID = args[0]
	}

	// Fetched before deleting so the output can show what was removed.
	link, err := client.GetIssueLink(ctx, linkID)
	if err != nil {
		return fmt.Errorf("failed to get issue link: %w", err)
	}
	if link.InwardIssue == nil || link.OutwardIssue == nil {
		return fmt.Errorf("issue link %s has no inward or outward issue", linkID)
	}

	if err := client.DeleteIssueLink(ctx, linkID); err != nil {
		return fmt.Errorf("failed to delete issue link: %w", err)
	}

	return outputJSON(linkDeleteResult{
		ID:           linkID,
		Type:         link.Type.Name,
		InwardIssue:  link.InwardIssue.Key,
		OutwardIssue: link.OutwardIssue.Key,
		Relation:     fmt.Sprintf("%s %s %s", link.InwardIssue.Key, link.Type.Outward, link.OutwardIssue.Key),
	})
}

func runLinkList(cmd *cobra.Command, args []string) error {
	issueKey, err := extractIssueKey(args[0])
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)

	links, err := client.GetIssueLinks(context.Background(), issueKey)
	if err != nil {
		return fmt.Errorf("failed to list issue links: %w", err)
	}

	entries := make([]linkListEntry, 0, len(links))
	for _, l := range links {
		entries = append(entries, linkListEntry{IssueLink: l, Relation: describeLink(issueKey, l)})
	}
	return outputJSON(entries)
}

func runLinkTypes(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := jira.NewClient(cfg)

	types, err := client.ListIssueLinkTypes(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list issue link types: %w", err)
	}

	return outputJSON(types)
}
