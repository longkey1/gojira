# gojira

JIRA CLI tool written in Go.

## Installation

### From Source

```bash
go install github.com/longkey1/gojira@latest
```

### From Release

Download the binary from [Releases](https://github.com/longkey1/gojira/releases) page.

## Configuration

Set the following environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `JIRA_EMAIL` | Yes | Your JIRA account email |
| `JIRA_API_TOKEN` | Yes | JIRA API token (generate at https://id.atlassian.com/manage-profile/security/api-tokens) |
| `JIRA_BASE_URL` | No | JIRA base URL (default: `https://your-domain.atlassian.net`) |

## Commands

### list

List tickets matching a JQL query.

```bash
# List all issues assigned to you
gojira list --jql 'assignee = currentUser()'

# List with specific fields
gojira list --jql 'project = PROJ' --fields 'key,summary,status'

# Filter by status
gojira list --jql 'project = PROJ AND status != Done'
```

### get

Get a single ticket by key.

```bash
# Get all fields
gojira get PROJ-1234

# Get specific fields
gojira get PROJ-1234 --fields 'summary,status,assignee'
```

### sum

Sum numeric field values for tickets matching a JQL query.

```bash
# Sum story points
gojira sum --jql 'parent = PROJ-1234' --field customfield_12345

# Sum any custom numeric field
gojira sum --jql 'project = PROJ AND sprint = 123' --field customfield_12345
```

### fields

List all available JIRA fields.

```bash
gojira fields
```

## Flags

### Common Flags

| Flag | Description |
|------|-------------|
| `--jql` | JQL query string |
| `--fields` | Comma-separated list of fields (default: `*all`) |
| `--field` | Single field name (for sum command) |

## Output

All commands output JSON to stdout, following JIRA API response structure.

## Development

```bash
# Build
make build

# Run tests
go test ./...

# Release
make release
```

## License

MIT
