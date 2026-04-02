#!/bin/bash
# devkit safety hook — blocks or prompts on dangerous operations
# Runs on PreToolUse for Bash, Edit, and Write tools

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# --- Bash: dangerous commands ---
if [ "$TOOL_NAME" = "Bash" ]; then

  # BLOCK: catastrophic filesystem destruction (rm -rf / or rm -rf ~ but not git rm)
  if echo "$COMMAND" | grep -qE '(^|[;&|]\s*)rm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+|--force\s+)?(\/\s*$|\/\s+|~\s|~\/|\$HOME\b)'; then
    echo "BLOCKED: destructive rm targeting root or home directory" >&2
    exit 2
  fi

  # BLOCK: rm -rf . (wipe current directory)
  if echo "$COMMAND" | grep -qE 'rm\s+(-[a-zA-Z]*r[a-zA-Z]*f|(-[a-zA-Z]*f[a-zA-Z]*r))\s+\.\s*$'; then
    echo "BLOCKED: rm -rf . would wipe the current directory" >&2
    exit 2
  fi

  # ASK: force push to main/master
  if echo "$COMMAND" | grep -qE 'git\s+push\s+.*--force.*\s+(main|master)\b|git\s+push\s+.*\s+(main|master)\s+.*--force'; then
    jq -n '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: "Force push to main/master — this can destroy remote history"
      }
    }'
    exit 0
  fi

  # ASK: git reset --hard
  if echo "$COMMAND" | grep -qE 'git\s+reset\s+--hard'; then
    jq -n '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: "git reset --hard discards all uncommitted changes — are you sure?"
      }
    }'
    exit 0
  fi

  # ASK: git checkout -- . (discard all changes)
  if echo "$COMMAND" | grep -qE 'git\s+checkout\s+--\s+\.'; then
    jq -n '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: "git checkout -- . discards all unstaged changes"
      }
    }'
    exit 0
  fi

  # ASK: git clean -fd (remove untracked files)
  if echo "$COMMAND" | grep -qE 'git\s+clean\s+(-[a-zA-Z]*f|--force)'; then
    jq -n '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: "git clean removes untracked files permanently"
      }
    }'
    exit 0
  fi

  # ASK: delete main/master branch
  if echo "$COMMAND" | grep -qE 'git\s+branch\s+(-[a-zA-Z]*D[a-zA-Z]*)\s+(main|master)\b'; then
    jq -n '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: "Deleting main/master branch — this is almost certainly a mistake"
      }
    }'
    exit 0
  fi

  # BLOCK: database destruction without WHERE clause
  if echo "$COMMAND" | grep -qiE '(DROP\s+TABLE|DROP\s+DATABASE|TRUNCATE\s+TABLE)'; then
    echo "BLOCKED: destructive database operation (DROP/TRUNCATE)" >&2
    exit 2
  fi

  if echo "$COMMAND" | grep -qiE 'DELETE\s+FROM\s+\w+\s*$|DELETE\s+FROM\s+\w+\s*;'; then
    echo "BLOCKED: DELETE FROM without WHERE clause" >&2
    exit 2
  fi

  # BLOCK: disk/filesystem destruction
  if echo "$COMMAND" | grep -qE 'dd\s+if=.*of=/dev/|mkfs\.|format\s+/dev/'; then
    echo "BLOCKED: disk/filesystem destruction command" >&2
    exit 2
  fi

  # ASK: recursive chmod 777
  if echo "$COMMAND" | grep -qE 'chmod\s+(-[a-zA-Z]*R[a-zA-Z]*)\s+777'; then
    jq -n '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: "chmod -R 777 makes everything world-writable — security risk"
      }
    }'
    exit 0
  fi

  # ASK: sudo rm
  if echo "$COMMAND" | grep -qE 'sudo\s+rm\s'; then
    jq -n '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: "sudo rm — elevated privilege deletion, confirm this is intentional"
      }
    }'
    exit 0
  fi

fi

# --- Edit/Write: protected files ---
if [ "$TOOL_NAME" = "Edit" ] || [ "$TOOL_NAME" = "Write" ]; then

  # BLOCK: writing to private keys
  if echo "$FILE_PATH" | grep -qE '\.(pem|key|p12|pfx)$'; then
    echo "BLOCKED: cannot write to private key file: $FILE_PATH" >&2
    exit 2
  fi

  # ASK: editing secrets/credentials
  if echo "$FILE_PATH" | grep -qiE '\.env($|\.)|credentials|\.secret|\.token|oauth.*\.json|service.account.*\.json'; then
    jq -n --arg file "$FILE_PATH" '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: ("Editing sensitive file: " + $file + " — may contain secrets")
      }
    }'
    exit 0
  fi

fi

# All clear
exit 0
