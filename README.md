# tk: A Git-Backed Issue Tracker

A minimal CLI ticket management system designed for AI agents. This is a Go port of [wedow/ticket](https://github.com/wedow/ticket).

## Overview

`tk` stores issues as markdown files with YAML frontmatter in a `.tickets/` directory, enabling easy content searching without bloating context windows. Based on Unix philosophy principles, it provides:

- **Dependency tracking** with cycle detection
- **Atomic claims** to prevent race conditions
- **Partial ID matching** for quick access
- **Git-native version control** for all ticket data

## Installation

### Go Install

Requires Go 1.25+:

```bash
go install github.com/radutopala/ticket/cmd/tk@latest
```

### From Source

```bash
git clone https://github.com/radutopala/ticket.git
cd ticket
make install
```

Both methods install `tk` to your `$GOPATH/bin`.

### Build Locally

```bash
make build
./bin/tk --help
```

## Requirements

- Go 1.25+ (for building)
- `jq` (optional, for the `query` command filtering)
- `$EDITOR` environment variable (for the `edit` command)

## Agent Integration

### Claude Code Setup

Add to your project's `CLAUDE.md`:

```markdown
This project uses `tk` for ticket management. Run `tk` to see available commands.

Workflow:
- Use `tk ready` to find tickets ready to work on
- Use `tk start <id>` to claim a ticket before working on it
- Use `tk add-note <id> "progress update"` to document progress
- Use `tk close <id>` when complete
```

Optionally, add to `.claude/settings.local.json` to allow ticket commands:

```json
{
  "permissions": {
    "allow": [
      "Bash(tk *)"
    ]
  }
}
```

### Other AI Agents

Add to your `AGENTS.md` or system prompt:

```
This project uses a CLI ticket system. Run `tk` for help. Key commands:
- tk ready          List tickets ready to work on
- tk start <id>     Claim a ticket
- tk show <id>      View ticket details
- tk close <id>     Mark complete
```

## Command Reference

### Core Operations

| Command | Description |
|---------|-------------|
| `create [title]` | Create a new ticket (outputs ID) |
| `show <id>` | Display ticket details |
| `edit <id>` | Open ticket in $EDITOR |
| `start <id>` | Mark as in_progress |
| `close <id>` | Mark as closed |
| `reopen <id>` | Revert to open status |
| `status <id> <status>` | Update status (open\|in_progress\|closed) |

### Create Options

```bash
tk create "My ticket title" \
  -d "Description text" \
  --design "Design notes" \
  --acceptance "Acceptance criteria" \
  -t feature \           # bug|feature|task|epic|chore (default: task)
  -p 1 \                 # Priority 0-4, 0=highest (default: 2)
  -a "John Doe" \        # Assignee (defaults to git user.name)
  --external-ref gh-123 \# External reference (e.g., JIRA-456)
  --parent tic-abc1 \    # Parent ticket ID
  --tags backend,urgent  # Comma-separated tags
```

### Dependency Management

| Command | Description |
|---------|-------------|
| `dep add <id> <dep-id>` | Add dependency (id depends on dep-id) |
| `dep remove <id> <dep-id>` | Remove dependency |
| `dep tree [id]` | Display dependency hierarchy |
| `dep tree --full` | Show full tree for all tickets |
| `dep check` | Identify circular dependencies |
| `undep <id> <dep-id>` | Alias for dep remove |

### Linking

| Command | Description |
|---------|-------------|
| `link <id> <id> [id...]` | Create symmetric links between tickets |
| `unlink <id> <target-id>` | Remove link between tickets |

### Listing & Filtering

| Command | Description |
|---------|-------------|
| `list` / `ls` | List all tickets |
| `ready` | Open/in_progress tickets with resolved deps |
| `blocked` | Open/in_progress tickets with unresolved deps |
| `closed` | Recently closed tickets |

All list commands support filters:
- `--status <status>` - Filter by status
- `-a, --assignee <name>` - Filter by assignee
- `-T, --tag <tag>` - Filter by tag
- `--limit <n>` - Limit results (closed command only, default: 20)

### Notes & Export

| Command | Description |
|---------|-------------|
| `add-note <id> [text]` | Append timestamped note (text or stdin) |
| `query [jq-filter]` | Export tickets as JSON, optionally filter with jq |

## Ticket Format

Tickets are stored as markdown files with YAML frontmatter:

```markdown
---
id: tic-a1b2
status: open
type: task
priority: 2
assignee: John Doe
tags:
  - backend
  - urgent
deps:
  - tic-c3d4
created: 2025-01-31T12:34:56Z
---
# Ticket Title

Description content here.

## Design

Design notes section.

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Notes

### 2025-01-31T14:00:00Z

Timestamped note content.
```

## Notable Features

### Partial ID Matching

Use any unique substring of a ticket ID:

```bash
tk show 5c4       # matches tic-5c46
tk start abc      # matches tic-abc1
```

### Directory Discovery

`tk` searches parent directories for `.tickets/`. Override with the `TICKETS_DIR` environment variable:

```bash
export TICKETS_DIR=/path/to/.tickets
```

### Pager Support

Output is automatically paged. Override with `TICKET_PAGER`:

```bash
export TICKET_PAGER=less
export TICKET_PAGER=cat  # disable paging
```

### Atomic Claims

The `start` command uses file locking to prevent race conditions when multiple agents claim tickets concurrently.

## Development

### Build

```bash
make build    # Build to bin/tk
make install  # Install to $GOPATH/bin
```

### Test

```bash
make test     # Run tests with race detection
```

### Lint

```bash
make lint     # Run golangci-lint
```

## Project Structure

```
.
├── cmd/tk/           # Main entry point
├── internal/
│   ├── cmd/          # CLI commands (Cobra)
│   ├── config/       # Configuration
│   ├── domain/       # Core data models
│   └── storage/      # File I/O operations
├── .tickets/         # Ticket storage directory
├── Makefile
└── go.mod
```

## Credits

This is a Go port of [wedow/ticket](https://github.com/wedow/ticket), originally a bash script. The Go implementation adds:

- Compiled binary for better performance
- Comprehensive test suite
- Atomic file operations with locking
- Structured logging

## License

MIT
