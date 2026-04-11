#!/usr/bin/env bash
set -euo pipefail

# devkit-stop-guard: Stop hook that blocks session end during active workflows.
# Outputs JSON: {"decision":"approve"} or {"decision":"block","reason":"..."}.
#
# Policy: symmetric with devkit-guard.sh. If the session file exists but
# cannot be parsed, we fail CLOSED (block with a clear reason) rather
# than silently approve — a corrupted state file during an active
# workflow is a bug the user needs to see, not something to paper over.
# Stale sessions (see lib/read-session.sh TTL) are treated as orphaned
# and the Stop is approved so the user is never wedged by a crashed
# engine process.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/read-session.sh
source "${SCRIPT_DIR}/lib/read-session.sh"

DATA_DIR="${CLAUDE_PLUGIN_DATA:-}"
if [[ -z "$DATA_DIR" ]]; then
  printf 'devkit-stop-guard: CLAUDE_PLUGIN_DATA unset — enforcement disabled\n' >&2
  printf '{"decision":"approve"}'
  exit 0
fi

SESSION_FILE="${DATA_DIR}/session.json"

if ! parse_session_fields "$SESSION_FILE"; then
  if [[ -f "$SESSION_FILE" ]]; then
    printf 'devkit-stop-guard: cannot parse session state (python3 missing or JSON corrupt); blocking Stop\n' >&2
    printf '{"decision":"block","reason":"devkit session state corrupted — remove %s to clear"}' "$SESSION_FILE"
    exit 0
  fi
  printf '{"decision":"approve"}'
  exit 0
fi

if [[ "$SESSION_STATUS" != "running" ]]; then
  printf '{"decision":"approve"}'
  exit 0
fi

if [[ "$SESSION_STALE" == "1" ]]; then
  printf 'devkit-stop-guard: session %s idle past TTL — approving Stop (reclaim on next devkit_start)\n' "$SESSION_WORKFLOW" >&2
  printf '{"decision":"approve"}'
  exit 0
fi

REMAINING=0
if [[ -n "$SESSION_TOTAL_STEPS" && -n "$SESSION_CURRENT_INDEX" ]]; then
  REMAINING=$((SESSION_TOTAL_STEPS - SESSION_CURRENT_INDEX))
fi
WF="${SESSION_WORKFLOW:-unknown}"

# Emit the block verdict via python3 json.dumps so the reason string is
# escaped safely — workflow names and step IDs flow from workflow YAML
# and could theoretically contain characters that need escaping.
python3 -c '
import json, sys
print(json.dumps({
    "decision": "block",
    "reason": f"Workflow {sys.argv[1]} incomplete — {sys.argv[2]} steps remaining. Call devkit_advance to continue."
}))
' "$WF" "$REMAINING"
exit 0
