#!/bin/bash
# devkit edit-time security hook — blocks known vulnerability patterns on Write/Edit
# Runs on PreToolUse for Edit and Write tools
#
# Catches security anti-patterns at the moment of creation rather than in a later review.
# Warns once per file+pattern per session to avoid spam.

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')
NEW_STRING=$(echo "$INPUT" | jq -r '.tool_input.new_string // .tool_input.content // empty')

# Only check Edit and Write
[ "$TOOL_NAME" = "Edit" ] || [ "$TOOL_NAME" = "Write" ] || exit 0
[ -z "$NEW_STRING" ] && exit 0

# Session dedup — warn once per file+pattern per session.
# $$ is the hook's own bash PID, which is brand new for every invocation
# (Claude Code spawns a fresh bash per hook call), so using it silently
# disabled dedup: every call wrote to a unique file and never found a
# prior warning. $PPID is the parent — the Claude Code hook runner — which
# is stable for the entire session, matching the "per session" intent.
SEEN_FILE="${TMPDIR:-/tmp}/devkit-security-seen-${PPID}"

check_pattern() {
  local pattern="$1"
  local message="$2"
  local key="${FILE_PATH}:${pattern}"

  if echo "$NEW_STRING" | grep -qE "$pattern"; then
    # Skip if already warned for this file+pattern
    if [ -f "$SEEN_FILE" ] && grep -qF "$key" "$SEEN_FILE" 2>/dev/null; then
      return
    fi
    echo "$key" >> "$SEEN_FILE" 2>/dev/null

    jq -n --arg reason "$message" '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "ask",
        permissionDecisionReason: $reason
      }
    }'
    exit 0
  fi
}

# --- JavaScript / TypeScript ---
if echo "$FILE_PATH" | grep -qE '\.(js|jsx|ts|tsx|mjs|cjs)$'; then
  check_pattern 'eval\s*\(' "Security: eval() enables code injection — use safer alternatives"
  check_pattern 'new\s+Function\s*\(' "Security: new Function() is equivalent to eval — avoid dynamic code generation"
  check_pattern 'dangerouslySetInnerHTML' "Security: dangerouslySetInnerHTML enables XSS — sanitize input with DOMPurify"
  check_pattern 'document\.write\s*\(' "Security: document.write enables XSS — use DOM APIs instead"
  check_pattern '\.innerHTML\s*=' "Security: innerHTML assignment enables XSS — use textContent or sanitize"
  check_pattern 'child_process.*\.exec\s*\(' "Security: child_process.exec is vulnerable to shell injection — use execFile or spawn"
  check_pattern 'crypto\.createHash\s*\(\s*["\x27]md5' "Security: MD5 is cryptographically broken — use SHA-256 or better"
  check_pattern 'crypto\.createHash\s*\(\s*["\x27]sha1' "Security: SHA-1 is deprecated — use SHA-256 or better"
  check_pattern 'Math\.random\s*\(' "Security: Math.random is not cryptographically secure — use crypto.randomUUID or crypto.getRandomValues"
fi

# --- Python ---
if echo "$FILE_PATH" | grep -qE '\.py$'; then
  check_pattern 'eval\s*\(' "Security: eval() enables code injection — use ast.literal_eval for data parsing"
  check_pattern 'exec\s*\(' "Security: exec() enables arbitrary code execution — avoid or sandbox"
  check_pattern 'pickle\.load' "Security: pickle.load executes arbitrary code — use json or msgpack for untrusted data"
  check_pattern 'os\.system\s*\(' "Security: os.system is vulnerable to shell injection — use subprocess.run with shell=False"
  check_pattern 'subprocess.*shell\s*=\s*True' "Security: shell=True enables shell injection — use shell=False with argument list"
  check_pattern '__import__\s*\(' "Security: __import__ with user input enables arbitrary module loading"
  check_pattern 'yaml\.load\s*\(' "Security: yaml.load executes arbitrary code — use yaml.safe_load"
  check_pattern 'hashlib\.(md5|sha1)\s*\(' "Security: MD5/SHA-1 are cryptographically broken — use SHA-256 or better"
fi

# --- Go ---
if echo "$FILE_PATH" | grep -qE '\.go$'; then
  check_pattern 'fmt\.Sprintf\s*\(.*%s.*\+' "Security: string concatenation in SQL/commands — use parameterized queries"
  check_pattern 'exec\.Command\s*\(\s*"(sh|bash)"' "Security: shell execution via exec.Command — pass arguments directly, avoid sh -c"
  check_pattern 'md5\.New\s*\(' "Security: MD5 is cryptographically broken — use SHA-256 or better"
  check_pattern 'sha1\.New\s*\(' "Security: SHA-1 is deprecated — use SHA-256 or better"
  check_pattern 'filepath\.(Join|Clean)\s*\([^)]*\b(name|input|arg|param)\b' "Security: filepath with user input — validate against path traversal (e.g., ^[a-zA-Z0-9_-]+$)"
fi

# --- SQL patterns (any file) ---
check_pattern "'\s*\+\s*\w+\s*\+\s*'" "Security: string concatenation in SQL — use parameterized queries to prevent SQL injection"

# --- Secrets in code (any file) ---
check_pattern '(password|secret|api_key|apikey|api_secret|access_token)\s*=\s*["\x27][^"\x27]{8,}' "Security: possible hardcoded secret — use environment variables or a secrets manager"

# All clear
exit 0
