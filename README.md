# gojira

A command-line interface tool for JIRA, written in Go.

## Installation

### From Source

```bash
go install github.com/longkey1/gojira@latest
```

### From Release

Download the binary from [Releases](https://github.com/longkey1/gojira/releases) page.

## Configuration

Set the following environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JIRA_EMAIL` | Yes | - | Your JIRA account email |
| `JIRA_API_TOKEN` | Yes | - | JIRA API token |
| `JIRA_BASE_URL` | No | `https://your-domain.atlassian.net` | JIRA instance base URL |

To generate an API token, visit: https://id.atlassian.com/manage-profile/security/api-tokens

### Environment Variable Expansion

Environment variable values support `${VAR}` or `$VAR` syntax for referencing other environment variables:

```bash
# Example: Store credentials separately and reference them
export MY_JIRA_EMAIL="user@example.com"
export MY_JIRA_TOKEN="your-api-token"

export JIRA_EMAIL='${MY_JIRA_EMAIL}'
export JIRA_API_TOKEN='${MY_JIRA_TOKEN}'
```

This is useful when managing credentials through secret managers or shared configuration files.

## Commands

### list

List issues matching a JQL query.

```bash
# List all issues assigned to you
gojira list --jql 'assignee = currentUser()'

# List with specific fields
gojira list --jql 'project = PROJ' --fields 'key,summary,status'

# Filter by status
gojira list --jql 'project = PROJ AND status != Done'
```

### get

Get a single issue by key.

```bash
# Get all fields
gojira get PROJ-1234

# Get specific fields
gojira get PROJ-1234 --fields 'summary,status,assignee'
```

### sum

Sum numeric field values for issues matching a JQL query. Non-numeric or null values are skipped with a warning.

```bash
# Sum a single field
gojira sum --jql 'project = PROJ AND sprint = 123' --fields customfield_12345

# Sum multiple fields
gojira sum --jql 'project = PROJ' --fields customfield_12345,customfield_67890
```

### fields

List all available JIRA fields.

```bash
gojira fields
```

## Flags

| Flag | Commands | Description |
|------|----------|-------------|
| `--jql` | list, sum | JQL query string |
| `--fields` | list, get, sum | Comma-separated list of fields |

## Output

All commands output JSON to stdout, following the JIRA API response structure.

## License

MIT
