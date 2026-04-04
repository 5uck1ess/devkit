#!/bin/bash
# devkit audit trail hook — logs all Bash commands with timestamps
# Runs on PreToolUse for Bash tool
#
# Log location: .devkit/audit.log (gitignored)
# Format: ISO-8601 timestamp | working directory | command

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Skip if no command
[ -z "$COMMAND" ] && exit 0

# Ensure log directory exists
LOG_DIR=".devkit"
LOG_FILE="${LOG_DIR}/audit.log"
mkdir -p "$LOG_DIR"

# Truncate command for log (first line only, max 500 chars)
CMD_SHORT=$(echo "$COMMAND" | head -1 | cut -c1-500)

# Append timestamped entry
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) | $(pwd) | ${CMD_SHORT}" >> "$LOG_FILE"

# Rotate if log exceeds 10k lines
if [ -f "$LOG_FILE" ] && [ "$(wc -l < "$LOG_FILE")" -gt 10000 ]; then
  tail -5000 "$LOG_FILE" > "${LOG_FILE}.tmp" && mv "${LOG_FILE}.tmp" "$LOG_FILE"
fi

# Always allow — this hook is observational only
exit 0
