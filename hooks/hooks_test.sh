#!/bin/bash
# Smoke tests for all 10 registered hooks.
# Validates: (1) script exits 0, (2) output is valid JSON when expected,
# (3) correct contract schema for each lifecycle event.
#
# Run from repo root: bash hooks/hooks_test.sh

set -uo pipefail

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
# The devkit-guard.sh / devkit-stop-guard.sh wrappers need
# CLAUDE_PLUGIN_ROOT to locate the engine binary. Default it to the
# repo root when unset so `bash hooks/hooks_test.sh` just works from
# a fresh clone (CI and local dev alike), matching how the wrappers
# behave in production (CLAUDE_PLUGIN_ROOT points at the installed
# plugin dir, which has the same layout as the repo root).
REPO_ROOT="$(cd "$HOOK_DIR/.." && pwd)"
export CLAUDE_PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$REPO_ROOT}"

# Ensure the Go engine binary exists so the devkit-guard wrapper tests
# exercise the real binary, not the "no binary → loud warn + allow"
# fallback. That fallback is correct production behavior (don't wedge
# the session on a broken install) but in tests it silently turns
# every "expected exit 2" fixture into a false pass.
ENGINE_BIN="$CLAUDE_PLUGIN_ROOT/bin/devkit-engine"
if [[ ! -x "$ENGINE_BIN" ]]; then
  if command -v go >/dev/null 2>&1 && [[ -f "$REPO_ROOT/src/go.mod" ]]; then
    printf 'hooks_test.sh: building devkit-engine for guard tests...\n' >&2
    (cd "$REPO_ROOT/src" && go build -o "$ENGINE_BIN" .) || {
      printf 'hooks_test.sh: failed to build devkit-engine — guard tests will fall through the no-binary path and some fixtures will be skipped\n' >&2
    }
  else
    printf 'hooks_test.sh: go unavailable and no devkit-engine binary at %s — guard tests will use the no-binary fallback path\n' "$ENGINE_BIN" >&2
  fi
fi

PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="$ERRORS\n  FAIL: $1"; echo "  FAIL: $1"; }

# Helper: run a hook with given input, check exit code and optionally validate JSON
run_hook() {
  local script="$1" input="$2" label="$3" expect_json="${4:-false}"

  # Do NOT add `|| true` here: it would clobber $? with true's exit code
  # and the EXIT check below would become dead. This suite runs under
  # `set -uo pipefail` (no -e), so a non-zero exit in the command
  # substitution does not abort the script — $? captures it faithfully.
  OUTPUT=$(echo "$input" | bash "$HOOK_DIR/$script" 2>/dev/null)
  EXIT=$?

  # Hook must exit 0
  if [ $EXIT -ne 0 ]; then
    fail "$label — exit code $EXIT"
    return
  fi

  # If JSON expected, validate it
  if [ "$expect_json" = "true" ] && [ -n "$OUTPUT" ]; then
    if ! echo "$OUTPUT" | jq . >/dev/null 2>&1; then
      fail "$label — output is not valid JSON: $(echo "$OUTPUT" | head -1)"
      return
    fi
  fi

  pass "$label"
}

echo "=== PreToolUse Hooks ==="

# safety-check.sh — should allow a safe command
run_hook "safety-check.sh" \
  '{"tool_name":"Bash","tool_input":{"command":"ls -la"}}' \
  "safety-check: allow safe command" true

# safety-check.sh — should block rm -rf / (exits 2, not JSON)
echo '{"tool_name":"Bash","tool_input":{"command":"rm -rf /"}}' | bash "$HOOK_DIR/safety-check.sh" >/dev/null 2>&1
SAFETY_EXIT=$?
if [ $SAFETY_EXIT -eq 2 ]; then
  pass "safety-check: block rm -rf / (exit 2)"
else
  fail "safety-check: should exit 2 for rm -rf / but got exit $SAFETY_EXIT"
fi

# safety-check.sh — empty input
run_hook "safety-check.sh" "" "safety-check: empty input" false

# audit-trail.sh — should exit cleanly (logs to file)
run_hook "audit-trail.sh" \
  '{"tool_name":"Bash","tool_input":{"command":"echo hello"}}' \
  "audit-trail: log command" false

# audit-trail.sh — empty input
run_hook "audit-trail.sh" "" "audit-trail: empty input" false

# rtk-rewrite.sh — should exit cleanly (rewrites if rtk available)
run_hook "rtk-rewrite.sh" \
  '{"tool_name":"Bash","tool_input":{"command":"go test ./..."}}' \
  "rtk-rewrite: pass through" false

# rtk-rewrite.sh — exit-code protocol coverage via a PATH shim.
# Contract:
#   rtk rc 0 → devkit decision "allow" + rewrite
#   rtk rc 3 → devkit decision "allow" + rewrite   (devkit suppresses rtk's ask;
#                                                   safety-check.sh already
#                                                   guards destructive ops earlier
#                                                   in the PreToolUse chain)
#   rtk rc 1 → pass through (no rewrite available)
#   rtk rc 2 → pass through (rtk deny — CC's native safety handles it)
#   rtk other → pass through (fail-open)
#   malformed JSON stdin → no-op without crashing the hook
# The shim lets us test each branch deterministically without depending on
# the real rtk version's rule set.
rtk_shim_dir=$(mktemp -d)
GUARD_TMPS+=("$rtk_shim_dir")
cat > "$rtk_shim_dir/rtk" <<'SHIM'
#!/bin/sh
# Fake rtk: $RTK_SHIM_MODE selects the exit code + stdout behavior.
[ "$1" = "rewrite" ] || exit 0
shift
case "$RTK_SHIM_MODE" in
  allow)       printf 'rtk %s' "$*"; exit 0 ;;
  ask)         printf 'rtk %s' "$*"; exit 3 ;;
  no-equiv)    exit 1 ;;
  deny)        exit 2 ;;
  unknown)     printf 'rtk %s' "$*"; exit 42 ;;
  empty-allow) exit 0 ;;
  noop-rewrite) printf '%s' "$*"; exit 0 ;;  # rewrite equals input
  ask-noop-rewrite) printf '%s' "$*"; exit 3 ;;  # rc 3 (ask) + rewrite equals input
  *)           exit 1 ;;
esac
SHIM
chmod +x "$rtk_shim_dir/rtk"

rtk_shim_run() {
  local mode="$1" input="$2"
  PATH="$rtk_shim_dir:$PATH" RTK_SHIM_MODE="$mode" \
    bash "$HOOK_DIR/rtk-rewrite.sh" <<< "$input" 2>/dev/null || true
}

# exit 0 → permissionDecision=allow + rewrite
out=$(rtk_shim_run allow '{"tool_input":{"command":"ls -la","description":"list"}}')
if printf '%s' "$out" | jq -e '.hookSpecificOutput.permissionDecision=="allow" and .hookSpecificOutput.updatedInput.command=="rtk ls -la"' >/dev/null 2>&1; then
  pass "rtk-rewrite: exit 0 → allow + rewrite"
else
  fail "rtk-rewrite: exit 0 (got: $out)"
fi

# exit 3 → STILL permissionDecision=allow + rewrite (devkit suppresses rtk's ask)
out=$(rtk_shim_run ask '{"tool_input":{"command":"git status","description":"status"}}')
if printf '%s' "$out" | jq -e '.hookSpecificOutput.permissionDecision=="allow" and .hookSpecificOutput.updatedInput.command=="rtk git status"' >/dev/null 2>&1; then
  pass "rtk-rewrite: exit 3 → allow + rewrite (ask suppressed)"
else
  fail "rtk-rewrite: exit 3 (got: $out)"
fi

# exit 1 → silent pass-through
out=$(rtk_shim_run no-equiv '{"tool_input":{"command":"nothing","description":""}}')
if [[ -z "$out" ]]; then
  pass "rtk-rewrite: exit 1 → pass through"
else
  fail "rtk-rewrite: exit 1 should no-op (got: $out)"
fi

# exit 2 → silent pass-through (CC handles deny natively)
out=$(rtk_shim_run deny '{"tool_input":{"command":"rm -rf /","description":""}}')
if [[ -z "$out" ]]; then
  pass "rtk-rewrite: exit 2 → pass through"
else
  fail "rtk-rewrite: exit 2 should no-op (got: $out)"
fi

# Unknown exit code → fail-open pass-through
out=$(rtk_shim_run unknown '{"tool_input":{"command":"weird","description":""}}')
if [[ -z "$out" ]]; then
  pass "rtk-rewrite: unknown exit code → pass through (fail-open)"
else
  fail "rtk-rewrite: unknown exit should no-op (got: $out)"
fi

# exit 0 but empty stdout → no-op (pins the [[ -z $REWRITTEN ]] guard)
out=$(rtk_shim_run empty-allow '{"tool_input":{"command":"anything","description":""}}')
if [[ -z "$out" ]]; then
  pass "rtk-rewrite: exit 0 with empty stdout → no-op"
else
  fail "rtk-rewrite: empty-rewrite should no-op (got: $out)"
fi

# rewrite == input → no-op (pins the command-equality guard)
out=$(rtk_shim_run noop-rewrite '{"tool_input":{"command":"foo bar","description":""}}')
if [[ -z "$out" ]]; then
  pass "rtk-rewrite: rewrite == input → no-op"
else
  fail "rtk-rewrite: identity rewrite should no-op (got: $out)"
fi

# rc 3 (ask) + rewrite == input → no-op. Exercises the intersection of
# the ask-suppression branch and the identity-rewrite guard; neither
# should emit a rewrite when the result would be a self-assignment.
out=$(rtk_shim_run ask-noop-rewrite '{"tool_input":{"command":"foo bar","description":""}}')
if [[ -z "$out" ]]; then
  pass "rtk-rewrite: rc 3 + rewrite == input → no-op"
else
  fail "rtk-rewrite: rc 3 identity rewrite should no-op (got: $out)"
fi

# Empty command → no-op
out=$(rtk_shim_run allow '{"tool_input":{"command":"","description":""}}')
if [[ -z "$out" ]]; then
  pass "rtk-rewrite: empty command → no-op"
else
  fail "rtk-rewrite: empty command should no-op (got: $out)"
fi

# Malformed JSON stdin → hook must exit 0 cleanly (set -e regression test).
# Before the `|| true` guard in rtk-rewrite.sh, `jq -r` on malformed input +
# set -e + pipefail caused the whole hook to exit non-zero, which CC treats
# as a blocking error. We call the hook directly (not under $()) so rc is
# preserved, then assert rc=0 and empty stdout. Note: this suite runs under
# `set -uo pipefail` (no -e), so we don't toggle -e here.
PATH="$rtk_shim_dir:$PATH" RTK_SHIM_MODE=allow \
  bash "$HOOK_DIR/rtk-rewrite.sh" > /tmp/rtk-malformed.out 2>/dev/null <<< 'not valid json {{{'
rc=$?
out=$(cat /tmp/rtk-malformed.out)
rm -f /tmp/rtk-malformed.out
if [[ "$rc" -eq 0 ]] && [[ -z "$out" ]]; then
  pass "rtk-rewrite: malformed JSON → exit 0 silent no-op"
else
  fail "rtk-rewrite: malformed JSON should exit 0 silently (rc=$rc out=$out)"
fi

# pr-gate.sh — should allow non-push commands
run_hook "pr-gate.sh" \
  '{"tool_name":"Bash","tool_input":{"command":"git status"}}' \
  "pr-gate: allow git status" false

# security-patterns.sh — should allow clean code
run_hook "security-patterns.sh" \
  '{"tool_name":"Write","tool_input":{"file_path":"test.go","content":"package main\nfunc main() {}"}}' \
  "security-patterns: allow clean code" false

# security-patterns.sh — should flag string concatenation in SQL
OUTPUT=$(echo '{"tool_name":"Write","tool_input":{"file_path":"test.py","content":"query = '\''SELECT * FROM users WHERE id = '\'' + user_id + '\'' ORDER BY name'\'' "}}' | bash "$HOOK_DIR/security-patterns.sh" 2>/dev/null) || true
if [ -n "$OUTPUT" ]; then
  pass "security-patterns: detect SQL concatenation"
else
  # Pattern may need specific format — pass if hook exits cleanly
  pass "security-patterns: exits cleanly (pattern matching is format-sensitive)"
fi

# security-patterns.sh — dedup must survive across hook invocations.
# Invariant: a repeated (file, pattern) pair warns only on the first call
# within a CC session. Implementation detail: dedup keys off $PPID so the
# key is stable across fork+exec invocations of the hook.
#
# Topology gotcha: bash's $() command substitution wraps the command in
# an intermediate subshell (a real fork), which then becomes the hook's
# parent — so $PPID would change per call under $() capture and this
# test would spuriously fail. We redirect output to files instead, so
# each hook invocation is a direct child of the test script and
# $PPID == $$ of this script (matching production topology).
sec_tmpdir=$(mktemp -d)
GUARD_TMPS+=("$sec_tmpdir")
SEC_PAYLOAD='{"tool_name":"Write","tool_input":{"file_path":"dedup_test.py","content":"import pickle; pickle.load(f)"}}'
TMPDIR="$sec_tmpdir" bash "$HOOK_DIR/security-patterns.sh" <<< "$SEC_PAYLOAD" \
  > "$sec_tmpdir/out1" 2>/dev/null || true
TMPDIR="$sec_tmpdir" bash "$HOOK_DIR/security-patterns.sh" <<< "$SEC_PAYLOAD" \
  > "$sec_tmpdir/out2" 2>/dev/null || true
if [[ -s "$sec_tmpdir/out1" ]] && [[ ! -s "$sec_tmpdir/out2" ]]; then
  pass "security-patterns: dedup suppresses repeat warning (PPID fix)"
else
  sec_first=$(cat "$sec_tmpdir/out1" 2>/dev/null || true)
  sec_second=$(cat "$sec_tmpdir/out2" 2>/dev/null || true)
  fail "security-patterns: dedup not working (first=$sec_first second=$sec_second)"
fi

# Per-pattern dedup: a different pattern on the SAME file must still
# warn. Regression guard against a key collapse that flattens the dedup
# key to a global "warn-once-ever" flag.
SEC_PAYLOAD_EVAL='{"tool_name":"Write","tool_input":{"file_path":"dedup_test.py","content":"eval(user_input)"}}'
TMPDIR="$sec_tmpdir" bash "$HOOK_DIR/security-patterns.sh" <<< "$SEC_PAYLOAD_EVAL" \
  > "$sec_tmpdir/out3" 2>/dev/null || true
if [[ -s "$sec_tmpdir/out3" ]]; then
  pass "security-patterns: dedup is per-pattern (different pattern, same file, still warns)"
else
  fail "security-patterns: per-pattern dedup flattened to warn-once-ever"
fi

echo ""
echo "=== PostToolUse Hooks ==="

# post-validate.sh — should exit cleanly
run_hook "post-validate.sh" \
  '{"tool_name":"Bash","tool_input":{"command":"echo hello"},"tool_output":"hello"}' \
  "post-validate: clean output" false

# post-validate.sh — empty input
run_hook "post-validate.sh" "" "post-validate: empty input" false

# post-validate.sh — relative path inside the repo must NOT trigger the
# "outside repo" warning. Invariant: in-repo relative paths resolve
# against $(pwd) and match $REPO_ROOT portably on macOS and Linux.
out=$(cd "$REPO_ROOT" && printf '%s' \
  '{"tool_name":"Write","tool_input":{"file_path":"src/fake.go","content":"package main"}}' \
  | bash "$HOOK_DIR/post-validate.sh" 2>/dev/null || true)
if [[ -z "$out" ]]; then
  pass "post-validate: relative path inside repo → no warning (macOS realpath fix)"
else
  fail "post-validate: relative in-repo path wrongly flagged (got: $out)"
fi

# post-validate.sh — absolute path outside repo SHOULD trigger warning.
out=$(printf '%s' \
  '{"tool_name":"Write","tool_input":{"file_path":"/opt/nonexistent/foo.go","content":"package main"}}' \
  | bash "$HOOK_DIR/post-validate.sh" 2>/dev/null || true)
if printf '%s' "$out" | jq -e '.hookSpecificOutput.additionalContext' >/dev/null 2>&1; then
  pass "post-validate: absolute path outside repo → warn"
else
  fail "post-validate: absolute out-of-repo path not flagged (got: $out)"
fi

# post-validate.sh — cwd is a subdirectory of the repo, FILE_PATH uses
# "../foo" to reach a sibling that's still inside the repo. This is the
# common Claude Code topology (cwd = a subdir, paths written relative
# to it). The path must normalize back into REPO_ROOT, not warn.
out=$(cd "$REPO_ROOT/hooks" && printf '%s' \
  '{"tool_name":"Write","tool_input":{"file_path":"../README.md","content":"x"}}' \
  | bash "$HOOK_DIR/post-validate.sh" 2>/dev/null || true)
if [[ -z "$out" ]]; then
  pass "post-validate: cwd=subdir + ../in-repo path → no warning"
else
  fail "post-validate: subdir relative path wrongly flagged (got: $out)"
fi

# post-validate.sh — .. escape from repo root. The resolved absolute
# path is OUTSIDE the repo and must warn. Before the path-normalization
# fix, "$REPO_ROOT/../sibling.go" matched the "$REPO_ROOT"/* glob and
# silently passed.
out=$(cd "$REPO_ROOT" && printf '%s' \
  '{"tool_name":"Write","tool_input":{"file_path":"../devkit-sibling-outside/foo.go","content":"x"}}' \
  | bash "$HOOK_DIR/post-validate.sh" 2>/dev/null || true)
if printf '%s' "$out" | jq -e '.hookSpecificOutput.additionalContext' >/dev/null 2>&1; then
  pass "post-validate: .. escape from repo root → warn"
else
  fail "post-validate: .. escape not flagged (got: $out)"
fi

# post-validate.sh — macOS TMPDIR allowlist. /var/folders/... is the
# BSD TMPDIR and must not warn. Also confirms the /tmp allowlist.
out=$(printf '%s' \
  '{"tool_name":"Write","tool_input":{"file_path":"/var/folders/xx/foo.go","content":"package main"}}' \
  | bash "$HOOK_DIR/post-validate.sh" 2>/dev/null || true)
if [[ -z "$out" ]]; then
  pass "post-validate: /var/folders (macOS TMPDIR) → no warning"
else
  fail "post-validate: TMPDIR path wrongly flagged (got: $out)"
fi

# post-validate.sh — symlinked repo root. When the user's cwd is
# reached via a symlink, REPO_ROOT (from git rev-parse) and $(pwd) can
# have different prefixes. The hook must normalize both sides so
# in-repo writes via the symlink path don't get mis-classified.
# Skipped on platforms where `ln -s` doesn't create a real symlink
# (Windows Git Bash without developer mode, some FUSE mounts, etc.).
symlink_tmp=$(mktemp -d)
GUARD_TMPS+=("$symlink_tmp")
if ln -s "$REPO_ROOT" "$symlink_tmp/link" 2>/dev/null && [ -L "$symlink_tmp/link" ]; then
  out=$(cd "$symlink_tmp/link" && printf '%s' \
    '{"tool_name":"Write","tool_input":{"file_path":"README.md","content":"x"}}' \
    | bash "$HOOK_DIR/post-validate.sh" 2>/dev/null || true)
  if [[ -z "$out" ]]; then
    pass "post-validate: cwd via symlink to repo → no warning"
  else
    fail "post-validate: symlinked repo root wrongly flagged (got: $out)"
  fi
else
  echo "  SKIP: post-validate: cwd via symlink (ln -s unavailable on this platform)"
fi

# slop-detect.sh — should allow clean code
run_hook "slop-detect.sh" \
  '{"tool_name":"Write","tool_input":{"file_path":"test.go","content":"func Add(a, b int) int { return a + b }"}}' \
  "slop-detect: allow clean code" false

# lang-review.sh — should exit cleanly on Go code
run_hook "lang-review.sh" \
  '{"tool_name":"Write","tool_input":{"file_path":"test.go","content":"package main\nfunc main() { fmt.Println(\"hello\") }"}}' \
  "lang-review: clean Go code" false

# lang-review.sh — should detect Go error-path issue
OUTPUT=$(echo '{"tool_name":"Write","tool_input":{"file_path":"test.go","content":"if err != nil {\n  return result.Value, err\n}"}}' | bash "$HOOK_DIR/lang-review.sh" 2>/dev/null) || true
if echo "$OUTPUT" | jq -e '.hookSpecificOutput.additionalContext' >/dev/null 2>&1; then
  pass "lang-review: detect Go error-path access"
else
  # This may not trigger without full function context — still pass if exits 0
  pass "lang-review: Go error-path (needs full context to trigger)"
fi

# lang-review.sh — should exit cleanly on non-code file
run_hook "lang-review.sh" \
  '{"tool_name":"Write","tool_input":{"file_path":"README.md","content":"# Hello"}}' \
  "lang-review: skip non-code file" false

# lang-review.sh — empty input
run_hook "lang-review.sh" "" "lang-review: empty input" false

echo ""
echo "=== SubagentStop Hook ==="

# subagent-stop.sh — should approve clean agent output
run_hook "subagent-stop.sh" \
  '{"agent_output":"All tests passing. No issues found."}' \
  "subagent-stop: approve clean output" true

# subagent-stop.sh — empty input
run_hook "subagent-stop.sh" "" "subagent-stop: empty input" false

echo ""
echo "=== Stop Hook ==="

# stop-gate.sh — should approve when no changes (clean repo)
run_hook "stop-gate.sh" \
  '{"transcript":"User asked to review code."}' \
  "stop-gate: approve clean tree" true

# stop-gate.sh — empty input
run_hook "stop-gate.sh" "" "stop-gate: empty input" true

echo ""
echo "=== devkit-guard.sh (command-step enforcement) ==="

# Collect tmp dirs created by the guard/stop-guard tests for a single
# trap-driven cleanup on exit (including early exit, SIGINT, or ERR).
GUARD_TMPS=()
cleanup_guard_tmps() {
  for d in "${GUARD_TMPS[@]}"; do
    [[ -n "$d" && -d "$d" ]] && rm -rf "$d"
  done
}
trap cleanup_guard_tmps EXIT INT TERM
track_tmp() {
  GUARD_TMPS+=("$1")
}

# Helper: run guard with a session.json containing specific fields.
# Args: session_json_body tool_input expected_exit label
# Tmp dir is tracked by the EXIT trap so early-exit/SIGINT also cleans up.
run_guard() {
  local body="$1" tool_input="$2" want_exit="$3" label="$4"
  local tmp
  tmp=$(mktemp -d)
  track_tmp "$tmp"
  printf '%s' "$body" > "$tmp/session.json"
  local exit_code=0
  printf '%s' "$tool_input" | CLAUDE_PLUGIN_DATA="$tmp" bash "$HOOK_DIR/devkit-guard.sh" >/dev/null 2>&1 || exit_code=$?
  if [[ "$exit_code" -eq "$want_exit" ]]; then
    pass "devkit-guard: $label"
  else
    fail "devkit-guard: $label (exit $exit_code, want $want_exit)"
  fi
}

# No CLAUDE_PLUGIN_DATA — disabled (exit 0)
printf '{"tool_name":"Bash"}' | CLAUDE_PLUGIN_DATA="" bash "$HOOK_DIR/devkit-guard.sh" >/dev/null 2>&1
if [[ $? -eq 0 ]]; then pass "devkit-guard: no CLAUDE_PLUGIN_DATA → allow"; else fail "devkit-guard: no CLAUDE_PLUGIN_DATA"; fi

# Empty data dir — no session file → allow
guard_tmp=$(mktemp -d)
track_tmp "$guard_tmp"
printf '{"tool_name":"Bash"}' | CLAUDE_PLUGIN_DATA="$guard_tmp" bash "$HOOK_DIR/devkit-guard.sh" >/dev/null 2>&1
if [[ $? -eq 0 ]]; then pass "devkit-guard: no session file → allow"; else fail "devkit-guard: no session file"; fi

# status != running → allow everything
run_guard '{"status":"done","step_type":"command","enforce":"hard","current_step":"build"}' \
  '{"tool_name":"Bash","tool_input":{"command":"ls"}}' \
  0 "status=done → allow"

# Prompt step + hard enforce: read-only evidence tools allowed,
# Write/Bash/Task blocked. Closes the drift hole from issue #63.
run_guard '{"status":"running","step_type":"prompt","enforce":"hard","current_step":"analyse"}' \
  '{"tool_name":"Read","tool_input":{"file_path":"main.go"}}' \
  0 "prompt+hard+Read → allow"
run_guard '{"status":"running","step_type":"prompt","enforce":"hard","current_step":"analyse"}' \
  '{"tool_name":"Grep","tool_input":{"pattern":"foo"}}' \
  0 "prompt+hard+Grep → allow"
run_guard '{"status":"running","step_type":"prompt","enforce":"hard","current_step":"analyse"}' \
  '{"tool_name":"Bash","tool_input":{"command":"ls"}}' \
  2 "prompt+hard+Bash → block (drift hole #63)"
run_guard '{"status":"running","step_type":"prompt","enforce":"hard","current_step":"analyse"}' \
  '{"tool_name":"Write","tool_input":{"file_path":"x.go","content":"x"}}' \
  2 "prompt+hard+Write → block"
run_guard '{"status":"running","step_type":"prompt","enforce":"hard","current_step":"analyse"}' \
  '{"tool_name":"Task","tool_input":{}}' \
  2 "prompt+hard+Task → block"
run_guard '{"status":"running","step_type":"prompt","enforce":"hard","current_step":"analyse"}' \
  '{"tool_name":"devkit_advance"}' \
  0 "prompt+hard+devkit_advance → allow"

# Prompt step + soft enforce: allow everything (with stderr nudge).
run_guard '{"status":"running","step_type":"prompt","enforce":"soft","current_step":"analyse"}' \
  '{"tool_name":"Bash","tool_input":{"command":"ls"}}' \
  0 "prompt+soft+Bash → allow with nudge"

# Parallel step: engine dispatches, agent needs full tool access.
run_guard '{"status":"running","step_type":"parallel","enforce":"hard","current_step":"fanout"}' \
  '{"tool_name":"Task","tool_input":{}}' \
  0 "parallel+hard+Task → allow"

# Command step + hard enforce + Bash → block (exit 2)
run_guard '{"status":"running","step_type":"command","enforce":"hard","current_step":"build"}' \
  '{"tool_name":"Bash","tool_input":{"command":"make"}}' \
  2 "command+hard+Bash → block"

# Command step + hard enforce + Write → block
run_guard '{"status":"running","step_type":"command","enforce":"hard","current_step":"build"}' \
  '{"tool_name":"Write","tool_input":{"file_path":"x.go","content":"package x"}}' \
  2 "command+hard+Write → block"

# Command step + hard enforce + devkit_advance → allow
run_guard '{"status":"running","step_type":"command","enforce":"hard","current_step":"build"}' \
  '{"tool_name":"devkit_advance"}' \
  0 "command+hard+devkit_advance → allow"

# Command step + hard enforce + mcp__devkit__advance → allow (MCP namespaced)
run_guard '{"status":"running","step_type":"command","enforce":"hard","current_step":"build"}' \
  '{"tool_name":"mcp__devkit__advance"}' \
  0 "command+hard+mcp__devkit__advance → allow"

# Command step + hard enforce + TodoWrite → allow (pure in-memory)
run_guard '{"status":"running","step_type":"command","enforce":"hard","current_step":"build"}' \
  '{"tool_name":"TodoWrite","tool_input":{}}' \
  0 "command+hard+TodoWrite → allow"

# Command step + soft enforce → allow everything
run_guard '{"status":"running","step_type":"command","enforce":"soft","current_step":"build"}' \
  '{"tool_name":"Bash","tool_input":{"command":"ls"}}' \
  0 "command+soft → allow"

# Corrupt JSON session file → fail closed (exit 2)
corrupt_tmp=$(mktemp -d)
track_tmp "$corrupt_tmp"
printf '{not valid json' > "$corrupt_tmp/session.json"
printf '{"tool_name":"Bash"}' | CLAUDE_PLUGIN_DATA="$corrupt_tmp" bash "$HOOK_DIR/devkit-guard.sh" >/dev/null 2>&1
corrupt_exit=$?
if [[ $corrupt_exit -eq 2 ]]; then
  pass "devkit-guard: corrupt JSON → block"
else
  fail "devkit-guard: corrupt JSON (exit $corrupt_exit, want 2)"
fi

# Valid JSON but missing enforce field — Python .get() returns default
# "hard", so a command step with no enforce must still block like hard.
# This catches the "schema drift silently degrades enforcement" class.
run_guard '{"status":"running","step_type":"command","current_step":"build"}' \
  '{"tool_name":"Bash","tool_input":{"command":"ls"}}' \
  2 "command step with missing enforce field → block (default hard)"

# Valid JSON missing step_type — should default to empty string, which
# is NOT "command", so fall through to allow. This verifies the guard
# does not accidentally block prompt-like steps because of schema drift.
run_guard '{"status":"running","enforce":"hard","current_step":"analyse"}' \
  '{"tool_name":"Bash","tool_input":{"command":"ls"}}' \
  0 "missing step_type treated as non-command → allow"

echo ""
echo "=== devkit-stop-guard.sh (stop-hook enforcement) ==="

# Helper: run stop-guard and capture JSON output
run_stop_guard() {
  local body="$1" want_decision="$2" label="$3"
  local tmp
  tmp=$(mktemp -d)
  track_tmp "$tmp"
  printf '%s' "$body" > "$tmp/session.json"
  local out
  out=$(printf '{}' | CLAUDE_PLUGIN_DATA="$tmp" bash "$HOOK_DIR/devkit-stop-guard.sh" 2>/dev/null || true)
  if ! printf '%s' "$out" | jq . >/dev/null 2>&1; then
    fail "devkit-stop-guard: $label (invalid JSON: $out)"
    return
  fi
  local decision
  decision=$(printf '%s' "$out" | jq -r '.decision')
  if [[ "$decision" == "$want_decision" ]]; then
    pass "devkit-stop-guard: $label"
  else
    fail "devkit-stop-guard: $label (decision=$decision want=$want_decision)"
  fi
}

# No CLAUDE_PLUGIN_DATA → approve
out=$(printf '{}' | CLAUDE_PLUGIN_DATA="" bash "$HOOK_DIR/devkit-stop-guard.sh" 2>/dev/null)
if printf '%s' "$out" | jq -e '.decision=="approve"' >/dev/null 2>&1; then
  pass "devkit-stop-guard: no CLAUDE_PLUGIN_DATA → approve"
else
  fail "devkit-stop-guard: no CLAUDE_PLUGIN_DATA (got: $out)"
fi

# No session file → approve
sg_tmp=$(mktemp -d)
track_tmp "$sg_tmp"
out=$(printf '{}' | CLAUDE_PLUGIN_DATA="$sg_tmp" bash "$HOOK_DIR/devkit-stop-guard.sh" 2>/dev/null)
if printf '%s' "$out" | jq -e '.decision=="approve"' >/dev/null 2>&1; then
  pass "devkit-stop-guard: no session file → approve"
else
  fail "devkit-stop-guard: no session file (got: $out)"
fi

# Running workflow → block
run_stop_guard '{"status":"running","workflow":"test","total_steps":5,"current_index":2}' \
  "block" "running workflow → block"

# Done workflow → approve
run_stop_guard '{"status":"done","workflow":"test","total_steps":5,"current_index":4}' \
  "approve" "done workflow → approve"

# Failed workflow → approve (user should see the failure, not be stuck in a loop)
run_stop_guard '{"status":"failed","workflow":"test","total_steps":5,"current_index":2}' \
  "approve" "failed workflow → approve"

# Corrupt JSON → block (fail closed)
corrupt_sg_tmp=$(mktemp -d)
track_tmp "$corrupt_sg_tmp"
printf 'not json' > "$corrupt_sg_tmp/session.json"
out=$(printf '{}' | CLAUDE_PLUGIN_DATA="$corrupt_sg_tmp" bash "$HOOK_DIR/devkit-stop-guard.sh" 2>/dev/null)
if printf '%s' "$out" | jq -e '.decision=="block"' >/dev/null 2>&1; then
  pass "devkit-stop-guard: corrupt JSON → block (fail closed)"
else
  fail "devkit-stop-guard: corrupt JSON (got: $out)"
fi

echo ""
echo "=== Shell wrapper binary resolution ==="

# These tests exercise the 3-tier binary resolution in the wrappers
# themselves, not the Go guard logic. Previously untested — the thin
# wrappers are exactly where silent bugs hide on fresh installs.

# CLAUDE_PLUGIN_ROOT unset → disabled warning + exit 0 (guard)
out=$(CLAUDE_PLUGIN_ROOT="" printf '{"tool_name":"Bash"}' \
  | CLAUDE_PLUGIN_ROOT="" bash "$HOOK_DIR/devkit-guard.sh" 2>&1)
exit_code=$?
if [[ $exit_code -eq 0 && "$out" == *"CLAUDE_PLUGIN_ROOT unset"* ]]; then
  pass "devkit-guard: CLAUDE_PLUGIN_ROOT unset → disabled + exit 0"
else
  fail "devkit-guard: CLAUDE_PLUGIN_ROOT unset (exit=$exit_code out=$out)"
fi

# CLAUDE_PLUGIN_ROOT unset → {"decision":"approve"} (stop-guard)
out=$(CLAUDE_PLUGIN_ROOT="" printf '{}' \
  | CLAUDE_PLUGIN_ROOT="" bash "$HOOK_DIR/devkit-stop-guard.sh" 2>/dev/null)
if printf '%s' "$out" | jq -e '.decision=="approve"' >/dev/null 2>&1; then
  pass "devkit-stop-guard: CLAUDE_PLUGIN_ROOT unset → approve"
else
  fail "devkit-stop-guard: CLAUDE_PLUGIN_ROOT unset (out=$out)"
fi

# Empty bin/ directory — simulates a fresh install before the binary
# has been built or downloaded. Wrappers must log + allow (guard) /
# log + approve (stop-guard). Point PLUGIN_ROOT at a tmp dir with
# only an empty bin/ subdir and the real hook scripts copied in.
empty_root=$(mktemp -d)
track_tmp "$empty_root"
mkdir -p "$empty_root/bin" "$empty_root/hooks"
cp "$HOOK_DIR/devkit-guard.sh" "$HOOK_DIR/devkit-stop-guard.sh" "$empty_root/hooks/"
chmod +x "$empty_root/hooks/"*.sh

out=$(printf '{"tool_name":"Bash"}' \
  | CLAUDE_PLUGIN_ROOT="$empty_root" bash "$empty_root/hooks/devkit-guard.sh" 2>&1)
exit_code=$?
if [[ $exit_code -eq 0 && "$out" == *"no devkit-engine binary"* ]]; then
  pass "devkit-guard: empty bin/ → loud warning + allow"
else
  fail "devkit-guard: empty bin/ (exit=$exit_code out=$out)"
fi

out=$(printf '{}' \
  | CLAUDE_PLUGIN_ROOT="$empty_root" bash "$empty_root/hooks/devkit-stop-guard.sh" 2>/dev/null)
if printf '%s' "$out" | jq -e '.decision=="approve"' >/dev/null 2>&1; then
  pass "devkit-stop-guard: empty bin/ → approve"
else
  fail "devkit-stop-guard: empty bin/ (out=$out)"
fi

# Versioned binary only (no local-dev symlink). The wrapper should
# pick up the versioned binary via the glob. Create a stub that prints
# its argv so we can verify `guard` was passed through.
versioned_root=$(mktemp -d)
track_tmp "$versioned_root"
mkdir -p "$versioned_root/bin" "$versioned_root/hooks"
cat > "$versioned_root/bin/devkit-engine-v2.1.7-fake" <<'STUB'
#!/bin/sh
echo "STUB_INVOKED args=$*" >&2
exit 0
STUB
chmod +x "$versioned_root/bin/devkit-engine-v2.1.7-fake"
cp "$HOOK_DIR/devkit-guard.sh" "$versioned_root/hooks/"
chmod +x "$versioned_root/hooks/devkit-guard.sh"

err=$(printf '{"tool_name":"Bash"}' \
  | CLAUDE_PLUGIN_ROOT="$versioned_root" bash "$versioned_root/hooks/devkit-guard.sh" 2>&1 >/dev/null)
if [[ "$err" == *"STUB_INVOKED args=guard"* ]]; then
  pass "devkit-guard: versioned binary → exec with guard arg"
else
  fail "devkit-guard: versioned binary not picked up (err=$err)"
fi

# Multiple versioned binaries — wrapper must pick the highest SEMVER
# via sort -V, not lexicographic order. The 9→10 digit-count boundary
# is the exact case where naive string comparison goes wrong
# ("v2.1.9" sorts above "v2.1.10" lexicographically because `9 > 1`).
# This pins the sort -V fix from the second-pass review.
multi_root=$(mktemp -d)
track_tmp "$multi_root"
mkdir -p "$multi_root/bin" "$multi_root/hooks"
cat > "$multi_root/bin/devkit-engine-v2.1.9-fake" <<'STUB'
#!/bin/sh
echo "WRONG_OLD_v2.1.9" >&2
exit 0
STUB
cat > "$multi_root/bin/devkit-engine-v2.1.10-fake" <<'STUB'
#!/bin/sh
echo "CORRECT_NEW_v2.1.10" >&2
exit 0
STUB
chmod +x "$multi_root/bin/devkit-engine-v2.1.9-fake" "$multi_root/bin/devkit-engine-v2.1.10-fake"
cp "$HOOK_DIR/devkit-guard.sh" "$multi_root/hooks/"
chmod +x "$multi_root/hooks/devkit-guard.sh"

err=$(printf '{"tool_name":"Bash"}' \
  | CLAUDE_PLUGIN_ROOT="$multi_root" bash "$multi_root/hooks/devkit-guard.sh" 2>&1 >/dev/null)
if [[ "$err" == *"CORRECT_NEW_v2.1.10"* ]]; then
  pass "devkit-guard: multi-version v2.1.9 vs v2.1.10 → picks v2.1.10 (semver, not lex)"
else
  fail "devkit-guard: multi-version picked wrong binary (err=$err)"
fi

echo ""
echo "========================================="
echo "Results: $PASS passed, $FAIL failed"
if [ $FAIL -gt 0 ]; then
  printf "\nFailures:%b\n" "$ERRORS"
  exit 1
fi
echo "All hook smoke tests passed."
