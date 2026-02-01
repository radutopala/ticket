---
name: tk
description: Git-backed ticket management for AI agents
---

# tk - Ticket Management

A minimal CLI for tracking tickets as markdown files in `.tickets/`.

## Workflow

```bash
tk ready              # 1. Find available work
tk start <id>         # 2. Claim a ticket
tk show <id>          # 3. Read details
# ... do the work
tk close <id>         # 4. Mark complete
```

## Core Commands

| Command | Description |
|---------|-------------|
| `tk create "title"` | Create ticket (returns ID) |
| `tk show <id>` | View ticket details |
| `tk start <id>` | Claim ticket (set to in_progress) |
| `tk close <id>` | Complete ticket |
| `tk reopen <id>` | Revert to open |
| `tk edit <id>` | Open in $EDITOR |

## Listing & Filtering

```bash
tk list                      # All tickets
tk list --status open        # By status (open|in_progress|closed)
tk list -a alice             # By assignee
tk list -T urgent            # By tag
tk ready                     # Open tickets with resolved deps
tk blocked                   # Tickets with unresolved deps
tk closed                    # Recently closed
```

## Create Options

```bash
tk create "Fix login bug" \
  -d "Description text" \
  -t bug \                   # Type: bug|feature|task|epic|chore
  -p 1 \                     # Priority: 0-4 (0=highest)
  -a alice \                 # Assignee
  --parent <epic-id> \       # Parent ticket
  --tags ui,urgent           # Tags
```

## Dependencies

```bash
tk dep add <id> <depends-on>   # Add dependency
tk dep remove <id> <dep-id>    # Remove dependency
tk dep tree                    # Show dependency tree
tk dep check                   # Check for cycles
```

## Notes & Links

```bash
tk add-note <id> "Progress update"   # Add timestamped note
tk link <id1> <id2>                  # Link related tickets
tk unlink <id1> <id2>                # Remove link
```

## Export

```bash
tk query                    # JSON output
tk query '.[] | .title'     # With jq filter
```

## Tips

- Partial ID matching: `tk show a1b` matches `tic-a1b2c3`
- Tickets stored in `.tickets/` as markdown with YAML frontmatter
- Git-friendly: commit tickets with your code
