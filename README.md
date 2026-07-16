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

Credentials can be supplied via a TOML config file, environment variables, or a combination of both. The priority is **environment variables > config file**.

| Setting | Env Variable | Config Key | Description |
|---------|--------------|------------|-------------|
| Email | `JIRA_EMAIL` | `email` | Your JIRA account email |
| API Token | `JIRA_API_TOKEN` | `api_token` | JIRA API token |
| Base URL | `JIRA_BASE_URL` | `base_url` | JIRA instance base URL (e.g., `https://your-domain.atlassian.net`) |
| Read Only | `JIRA_READ_ONLY` | `read_only` | Block write commands (`create`/`update`/`comment add\|update\|delete`) |

To generate an API token, visit: https://id.atlassian.com/manage-profile/security/api-tokens

### Config File

A TOML config file is loaded from one of the following locations (first match wins):

1. Path given to `--config <path>`
2. `$XDG_CONFIG_HOME/gojira/config.toml`
3. `$HOME/.config/gojira/config.toml`

Example `config.toml`:

```toml
email = "user@example.com"
api_token = "your-api-token"
base_url = "https://your-domain.atlassian.net"
```

### Environment Variable Expansion

Values (in either env vars or the config file) support `${VAR}` or `$VAR` syntax for referencing other environment variables:

```bash
# Example: Store credentials separately and reference them
export MY_JIRA_EMAIL="user@example.com"
export MY_JIRA_TOKEN="your-api-token"

export JIRA_EMAIL='${MY_JIRA_EMAIL}'
export JIRA_API_TOKEN='${MY_JIRA_TOKEN}'
```

This is useful when managing credentials through secret managers or shared configuration files.

### Read-only Mode

Block write commands (`create`, `update`, `comment add`, `comment update`, `comment delete`) from running, before any JIRA API call is made. Useful when exposing the CLI to an LLM or other automation that should never write:

```bash
# Environment variable
export JIRA_READ_ONLY="true"

# Or config file (~/.config/gojira/config.toml)
read_only = true
```

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

# List with description as Markdown
gojira list --jql 'project = PROJ' --markdown-description
```

### get

Get a single issue by key or browse URL.

```bash
# Get all fields
gojira get PROJ-1234

# Get by browse URL
gojira get https://your-domain.atlassian.net/browse/PROJ-1234

# Get specific fields
gojira get PROJ-1234 --fields 'summary,status,assignee'

# Get with description as Markdown
gojira get PROJ-1234 --markdown-description
```

### create

Create a new issue.

```bash
# Create a task
gojira create --project PROJ --type Task --summary 'New task'

# Create a bug with priority and labels
gojira create --project PROJ --type Bug --summary 'Bug report' --priority 'High' --labels 'bug'

# Create with Markdown description
gojira create --project PROJ --type Story --summary 'Feature' --markdown-description --description '## Background
See [here](https://example.com) for details'

# Create a subtask
gojira create --project PROJ --type Sub-task --summary 'Subtask' --parent PROJ-123

# Create from a JSON file
gojira create --data-file ./new-issue.json
```

### update

Update an existing issue.

```bash
# Update summary
gojira update PROJ-123 --summary 'New title'

# Update description with ADF JSON
gojira update PROJ-123 --description '{"type":"doc","version":1,"content":[...]}'

# Update description with Markdown
gojira update PROJ-123 --markdown-description --description '## Overview
- Item 1
- Item 2'

# Update multiple fields
gojira update PROJ-123 --summary 'New title' --priority 'High' --labels 'bug,critical'

# Update with arbitrary JSON data
gojira update PROJ-123 --data '{"fields":{"customfield_10001":"value"}}'

# Use output from 'gojira get' as base
gojira get PROJ-123 > issue.json
gojira update PROJ-123 --data-file ./issue.json --summary 'Updated title'
```

### comment

Manage comments on an issue.

```bash
# List all comments on an issue
gojira comment list PROJ-123

# List comments with body as Markdown
gojira comment list PROJ-123 --markdown-body

# Add a comment (Markdown)
gojira comment add PROJ-123 --markdown-body --body 'This is a comment'

# Add a comment (ADF JSON)
gojira comment add PROJ-123 --body '{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}'

# Reply to an existing comment (quotes the original)
gojira comment add PROJ-123 --markdown-body --body 'I agree' --reply-to 10001

# Update a comment
gojira comment update PROJ-123 --comment-id 10001 --markdown-body --body 'Updated text'

# Delete a comment
gojira comment delete PROJ-123 --comment-id 10001
```

### fields

List all available JIRA fields.

```bash
gojira fields
```

### merge

Merge JSON files containing issues from a directory. When duplicate issues are found (same key), the one with the latest updated date is kept.

```bash
# Merge all JSON files in a directory
gojira merge --dir ./output

# Merge with specific file pattern
gojira merge --dir ./output --pattern 'issues-*.json'

# Merge recursively (search subdirectories)
gojira merge --dir ./output --recursive
```

### config

Get a single configuration value resolved from the config file and environment variables. Valid keys: `base_url`, `email`, `api_token`.

```bash
gojira config get base_url
gojira config get email
gojira config get api_token
```

### version

Show version information (version number, git commit SHA, build time, Go version).

```bash
# Full version info
gojira version

# Version number only
gojira version --short
```

## Flags

| Flag | Commands | Description |
|------|----------|-------------|
| `--config` | (global) | Path to TOML config file |
| `--jql` | list | JQL query string |
| `--fields` | list, get | Comma-separated list of fields |
| `--markdown-description` | get, list, create, update | Treat description as Markdown (input: convert to ADF, output: convert from ADF) |
| `--summary` | create, update | Issue summary |
| `--description` | create, update | Issue description (ADF JSON by default, Markdown with `--markdown-description`) |
| `--project` | create | Project key |
| `--type` | create | Issue type name (e.g., Task, Bug, Story) |
| `--assignee` | create, update | Assignee account ID |
| `--labels` | create, update | Labels (comma-separated) |
| `--priority` | create, update | Priority name |
| `--parent` | create | Parent issue key (for subtasks) |
| `--data` | create, update | Arbitrary JSON string for request body |
| `--data-file` | create, update | Path to JSON file for request body |
| `--body` | comment add, comment update | Comment body (ADF JSON by default, Markdown with `--markdown-body`) |
| `--markdown-body` | comment list, comment add, comment update | Treat comment body as Markdown (input: convert to ADF, output: convert from ADF) |
| `--comment-id` | comment update, comment delete | Comment ID |
| `--reply-to` | comment add | Comment ID to reply to (quotes the original comment) |
| `--dir` | merge | Directory to search for JSON files |
| `--pattern` | merge | File name pattern (glob, default: *.json) |
| `--recursive`, `-r` | merge | Search recursively in subdirectories |
| `--short`, `-s` | version | Show only the version number |

## Output

All commands output JSON to stdout, following the JIRA API response structure.

## License

MIT
