---
name: devkit:gcli
description: Google Workspace CLI (Gmail, Calendar, Drive) via gcli — use --for-ai flag for token-efficient structured output.
---

# gcli — Google Workspace CLI

Single binary for Gmail, Calendar, and Drive. Always use `--for-ai` for structured, token-efficient output.

## Prerequisites

Before using any gcli command, verify it is installed and authenticated:

```bash
command -v gcli >/dev/null 2>&1 && gcli mail list 1 --for-ai >/dev/null 2>&1 && echo "gcli: ready" || echo "gcli: not installed or not authenticated — install from https://github.com/AryaLabsHQ/gcli and run 'gcli login'"
```

If gcli is not installed, tell the user and stop. Do not attempt gcli commands without this check.

## Gmail

```bash
# List inbox (default 20)
gcli mail list --for-ai
gcli mail list 10 --unread --for-ai

# Read a thread
gcli mail get <thread-id> --for-ai

# Search
gcli mail search "from:boss subject:review" --for-ai

# Send
gcli mail send -t "user@example.com" -s "Subject" --body-file msg.txt

# Reply (use thread-id from list/search)
gcli mail reply <thread-id> --body-file reply.txt
gcli mail reply <thread-id> --all --body-file reply.txt

# Forward / mark
gcli mail forward <thread-id> -t "other@example.com"
gcli mail mark <message-id> --read
gcli mail mark <message-id> --trash
```

## Calendar

```bash
# Today's events
gcli cal list --for-ai

# Next 7 days
gcli cal list 7 --for-ai

# All calendars
gcli cal list 7 --all-calendars --for-ai

# Event details
gcli cal get <event-id> --for-ai

# Create
gcli cal create "Meeting" -s "tomorrow 2pm" -d "1h" --attendee "person@example.com"

# Edit / delete
gcli cal edit <event-id> -s "tomorrow 3pm"
gcli cal delete <event-id>
```

## Drive

```bash
# List root
gcli drive list --for-ai

# List folder
gcli drive list "path/to/folder" --for-ai

# Search
gcli drive search "quarterly report" --for-ai

# Shared files
gcli drive list -s --for-ai

# Download / upload
gcli drive download "path/to/file" -o ./local-file
gcli drive upload ./local-file "path/to/folder"

# Info / permissions
gcli drive info "path/to/file" --for-ai
gcli drive permissions "path/to/file" --for-ai
```

## Composing Emails

When writing email body:
1. Write content to a temp file
2. Use `--body-file` to send — avoids shell escaping issues
3. For replies, `--for-ai` reads stdin as body

## Tips

- Thread IDs from `mail list` and `mail search` feed into `mail get`, `mail reply`, `mail forward`
- Event IDs from `cal list` feed into `cal get`, `cal edit`, `cal delete`
- `--for-ai` strips HTML and formats for LLM consumption — always use it for reads
- Attachments: `-A file1.pdf -A file2.pdf` (repeatable)
