#!/usr/bin/env bash
set -euo pipefail

# Fixture matrix test for devkit-guard.sh.
# Seeds CLAUDE_PLUGIN_DATA with a crafted session.json and pipes a
# synthetic PreToolUse payload on stdin. Asserts exit code and the
# substring of whatever stderr diagnostic the guard emitted.
#
# Run: bash hooks/devkit-guard_test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GUARD="${SCRIPT_DIR}/devkit-guard.sh"

if [[ ! -x "$GUARD" ]]; then
  chmod +x "$GUARD" || true
fi

PASS=0
FAIL=0
FAILED_CASES=()

# run_case <label> <session-json|""> <tool-name> <expected-exit> <expected-stderr-substr-or-empty>
run_case() {
  local label="$1"
  local session_json="$2"
  local tool_name="$3"
  local expected_exit="$4"
  local expected_substr="${5:-}"

  local tmpdir
  tmpdir=$(mktemp -d)
  # shellcheck disable=SC2064
  trap "rm -rf '$tmpdir'" RETURN

  if [[ -n "$session_json" ]]; then
    printf '%s' "$session_json" > "${tmpdir}/session.json"
  fi

  local payload
  payload=$(printf '{"tool_name":"%s"}' "$tool_name")

  set +e
  CLAUDE_PLUGIN_DATA="$tmpdir" output=$(printf '%s' "$payload" | bash "$GUARD" 2>&1)
  local actual_exit=$?
  set -e

  local ok=1
  if [[ "$actual_exit" != "$expected_exit" ]]; then
    ok=0
  fi
  if [[ -n "$expected_substr" && "$output" != *"$expected_substr"* ]]; then
    ok=0
  fi

  if [[ $ok -eq 1 ]]; then
    PASS=$((PASS + 1))
    printf '  PASS  %s\n' "$label"
  else
    FAIL=$((FAIL + 1))
    FAILED_CASES+=("$label (exit=$actual_exit, expected=$expected_exit, output=$output)")
    printf '  FAIL  %s  (exit=%s expected=%s)\n    output: %s\n' "$label" "$actual_exit" "$expected_exit" "$output"
  fi

  rm -rf "$tmpdir"
  trap - RETURN
}

# --- Fixtures --------------------------------------------------------------

NOW_ISO=$(python3 -c 'import datetime; print(datetime.datetime.now(datetime.timezone.utc).isoformat())')
STALE_ISO="2020-01-01T00:00:00+00:00"

fresh_session() {
  local step_type="$1"
  local enforce="$2"
  local status="${3:-running}"
  printf '{
  "id": "abc123",
  "workflow": "tri-review",
  "current_step": "gather",
  "current_index": 0,
  "total_steps": 6,
  "step_type": "%s",
  "enforce": "%s",
  "status": "%s",
  "started_at": "%s",
  "updated_at": "%s"
}' "$step_type" "$enforce" "$status" "$NOW_ISO" "$NOW_ISO"
}

stale_session() {
  printf '{
  "id": "abc123",
  "workflow": "tri-review",
  "current_step": "gather",
  "current_index": 0,
  "total_steps": 6,
  "step_type": "prompt",
  "enforce": "hard",
  "status": "running",
  "started_at": "%s",
  "updated_at": "%s"
}' "$STALE_ISO" "$STALE_ISO"
}

# --- Cases -----------------------------------------------------------------

printf 'devkit-guard fixture matrix:\n'

# No session file → always allow.
run_case "no-session allows Write" "" "Write" 0 ""
run_case "no-session allows Bash"  "" "Bash"  0 ""

# status != running → always allow.
run_case "status=done allows Write" \
  "$(fresh_session prompt hard done)" "Write" 0 ""

# Command step + hard: devkit + TodoWrite allowed, rest blocked.
run_case "command-hard allows devkit_advance" \
  "$(fresh_session command hard)" "mcp__plugin_devkit_devkit-engine__devkit_advance" 0 ""
run_case "command-hard allows TodoWrite" \
  "$(fresh_session command hard)" "TodoWrite" 0 ""
run_case "command-hard blocks Write" \
  "$(fresh_session command hard)" "Write" 2 "BLOCKED"
run_case "command-hard blocks Bash" \
  "$(fresh_session command hard)" "Bash" 2 "BLOCKED"

# Prompt step + hard: evidence tools allowed, writes blocked. This is
# the new behaviour closing issue #63.
run_case "prompt-hard allows Read"  "$(fresh_session prompt hard)" "Read"  0 ""
run_case "prompt-hard allows Grep"  "$(fresh_session prompt hard)" "Grep"  0 ""
run_case "prompt-hard allows Glob"  "$(fresh_session prompt hard)" "Glob"  0 ""
run_case "prompt-hard allows TodoWrite" "$(fresh_session prompt hard)" "TodoWrite" 0 ""
run_case "prompt-hard allows devkit_advance" \
  "$(fresh_session prompt hard)" "mcp__plugin_devkit_devkit-engine__devkit_advance" 0 ""
run_case "prompt-hard blocks Write" "$(fresh_session prompt hard)" "Write" 2 "tri-review"
run_case "prompt-hard blocks Edit"  "$(fresh_session prompt hard)" "Edit"  2 "BLOCKED"
run_case "prompt-hard blocks Bash"  "$(fresh_session prompt hard)" "Bash"  2 "gather"
run_case "prompt-hard blocks Task"  "$(fresh_session prompt hard)" "Task"  2 "BLOCKED"
run_case "prompt-hard blocks WebFetch" \
  "$(fresh_session prompt hard)" "WebFetch" 2 "BLOCKED"

# Prompt step + soft: allow everything but emit a nudge on stderr.
run_case "prompt-soft allows Write with nudge" \
  "$(fresh_session prompt soft)" "Write" 0 "call devkit_advance"
run_case "prompt-soft allows Bash" \
  "$(fresh_session prompt soft)" "Bash" 0 "devkit-guard:"

# Parallel step: always allow.
run_case "parallel allows Write" \
  "$(fresh_session parallel hard)" "Write" 0 ""

# Stale session → allow + warn, regardless of step type. The stale
# branch fires BEFORE the step-type dispatch, so flipping the order
# would regress these cases.
run_case "stale-prompt-hard allows Write (orphaned)" \
  "$(stale_session)" "Write" 0 "orphaned"

stale_session_with() {
  # Args: step_type enforce
  printf '{
  "id": "abc123",
  "workflow": "tri-review",
  "current_step": "gather",
  "current_index": 0,
  "total_steps": 6,
  "step_type": "%s",
  "enforce": "%s",
  "status": "running",
  "started_at": "%s",
  "updated_at": "%s"
}' "$1" "$2" "$STALE_ISO" "$STALE_ISO"
}

run_case "stale-command-hard allows Bash (orphaned)" \
  "$(stale_session_with command hard)" "Bash" 0 "orphaned"
run_case "stale-prompt-soft allows Write (orphaned)" \
  "$(stale_session_with prompt soft)" "Write" 0 "orphaned"

# Backward-compat: session with only started_at (no updated_at) —
# the parser falls through to started_at for pre-UpdatedAt binaries.
run_case "legacy-session (no updated_at) treated as fresh when recent" \
  "$(printf '{
  "id": "legacy1",
  "workflow": "tri-review",
  "current_step": "gather",
  "current_index": 0,
  "total_steps": 6,
  "step_type": "prompt",
  "enforce": "hard",
  "status": "running",
  "started_at": "%s"
}' "$NOW_ISO")" "Write" 2 "BLOCKED"

# Corrupt JSON → fail-closed exit 2. Exercises the new code path on
# the shared helper; the legacy hooks_test.sh only covered the old
# inline parser.
corrupt_tmp=$(mktemp -d)
printf '{this is not json' > "$corrupt_tmp/session.json"
set +e
CLAUDE_PLUGIN_DATA="$corrupt_tmp" output=$(printf '{"tool_name":"Bash"}' | bash "$GUARD" 2>&1)
corrupt_exit=$?
set -e
if [[ "$corrupt_exit" == "2" && "$output" == *"Cannot parse session state"* ]]; then
  PASS=$((PASS + 1))
  printf '  PASS  corrupt JSON fails closed with diagnostic\n'
else
  FAIL=$((FAIL + 1))
  printf '  FAIL  corrupt JSON fails closed (exit=%s output=%s)\n' "$corrupt_exit" "$output"
fi
rm -rf "$corrupt_tmp"

# C1 regression guard: the devkit MCP allowlist must be narrow enough
# not to admit a third-party MCP tool whose name happens to contain
# "devkit". Use a plausible-looking foreign tool name.
run_case "prompt-hard blocks foreign devkit-like mcp" \
  "$(fresh_session prompt hard)" "mcp__other__my_devkit_plugin" 2 "BLOCKED"
run_case "prompt-hard allows real devkit-engine mcp" \
  "$(fresh_session prompt hard)" "mcp__plugin_devkit_devkit-engine__devkit_advance" 0 ""

# --- Report ---------------------------------------------------------------

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
if [[ $FAIL -gt 0 ]]; then
  printf '\nFailed cases:\n'
  for c in "${FAILED_CASES[@]}"; do
    printf '  - %s\n' "$c"
  done
  exit 1
fi
exit 0
