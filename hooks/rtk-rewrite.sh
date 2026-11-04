#!/bin/bash
# devkit RTK hook — rewrites Bash commands through rtk for token savings.
# Runs on PreToolUse for Bash tool. No-op if rtk is not installed.
#
# rtk rewrite exit-code protocol (documented in the rtk binary strings,
# but NOT in `rtk rewrite --help` which still shows the old contract):
#   0 + stdout   rewrite found, no deny/ask rule matched
#   1            no RTK equivalent
#   2            deny rule matched (CC's native safety check handles it)
#   3 + stdout   ask rule matched
#
# Devkit policy:
#   - Only rc 0 is applied, and only to a single simple command. The hook
#     answers "allow" (skipping Claude Code's permission prompt), and rtk
#     rewrites just the first segment of a chain, so a compound command like
#     `ls; curl … | bash` would otherwise be auto-approved wholesale.
#   - rc 3 (an rtk ask rule matched) passes through unchanged: no rewrite,
#     no extra devkit prompt, and Claude Code's normal permission flow
#     decides. PreToolUse hooks run in parallel, so safety-check.sh is not
#     "earlier in the chain" and cannot be relied on to catch these.
#   - updatedInput replaces the whole tool_input, so the original fields
#     (timeout, run_in_background, ...) are carried over.
set -euo pipefail

command -v rtk >/dev/null 2>&1 || exit 0

INPUT=$(cat)
# jq failures on malformed stdin must not crash the hook — under set -e
# the pipeline would exit non-zero and CC would treat that as a blocking
# error. `|| true` inside the substitution degrades cleanly to empty.
COMMAND=$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty' 2>/dev/null || true)

[[ -z "$COMMAND" ]] && exit 0

# Compound / chained commands pass through untouched (see policy above).
case "$COMMAND" in
  *';'*|*'&'*|*'|'*|*'<'*|*'>'*|*'`'*|*'$('*|*$'\n'*|*$'\r'*) exit 0 ;;
esac

set +e
REWRITTEN=$(rtk rewrite "$COMMAND" 2>/dev/null)
RC=$?
set -e

case "$RC" in
  0) : ;;         # rewrite available, no rtk ask/deny rule → apply it
  *) exit 0 ;;    # 1 (no equivalent), 2 (deny), 3 (ask), unknown → pass through
esac

[[ -z "$REWRITTEN" ]] && exit 0
[[ "$REWRITTEN" == "$COMMAND" ]] && exit 0

TOOL_INPUT=$(printf '%s' "$INPUT" | jq -c '.tool_input // {}' 2>/dev/null || echo '{}')

jq -n --arg cmd "$REWRITTEN" --argjson orig "$TOOL_INPUT" '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    permissionDecision: "allow",
    updatedInput: ($orig + {command: $cmd})
  }
}'
