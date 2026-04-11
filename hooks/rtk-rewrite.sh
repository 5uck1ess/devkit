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
# Devkit policy: both 0 and 3 auto-apply the rewrite. rtk's "ask" rules
# are noisy on common commands (ls, git, find) and devkit's safety-check.sh
# already fires earlier in the Bash PreToolUse chain for destructive ops,
# so a second per-command prompt from rtk is redundant.
set -euo pipefail

command -v rtk >/dev/null 2>&1 || exit 0

INPUT=$(cat)
# jq failures on malformed stdin must not crash the hook — under set -e
# the pipeline would exit non-zero and CC would treat that as a blocking
# error. `|| true` inside the substitution degrades cleanly to empty.
COMMAND=$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty' 2>/dev/null || true)

[[ -z "$COMMAND" ]] && exit 0

set +e
REWRITTEN=$(rtk rewrite "$COMMAND" 2>/dev/null)
RC=$?
set -e

case "$RC" in
  0|3) : ;;       # rewrite available (0) or available-but-ask (3) → apply it
  *)   exit 0 ;;  # 1 (no equivalent), 2 (deny handled natively), or unknown → pass through
esac

[[ -z "$REWRITTEN" ]] && exit 0
[[ "$REWRITTEN" == "$COMMAND" ]] && exit 0

DESCRIPTION=$(printf '%s' "$INPUT" | jq -r '.tool_input.description // empty' 2>/dev/null || true)

jq -n --arg cmd "$REWRITTEN" --arg desc "$DESCRIPTION" '{
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    permissionDecision: "allow",
    updatedInput: {
      command: $cmd,
      description: $desc
    }
  }
}'
