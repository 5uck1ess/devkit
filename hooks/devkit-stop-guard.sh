#!/usr/bin/env bash
set -euo pipefail

# devkit-stop-guard: Stop hook that blocks session end during active workflows.
# Outputs JSON: {"decision":"approve"} or {"decision":"block","reason":"..."}.
#
# Policy: symmetric with devkit-guard.sh. If the session file exists but
# cannot be parsed, we fail CLOSED (block with a clear reason) rather
# than silently approve — a corrupted state file during an active
# workflow is a bug the user needs to see, not something to paper over.

DATA_DIR="${CLAUDE_PLUGIN_DATA:-}"
if [[ -z "$DATA_DIR" ]]; then
  printf 'devkit-stop-guard: CLAUDE_PLUGIN_DATA unset — enforcement disabled\n' >&2
  printf '{"decision":"approve"}'
  exit 0
fi

SESSION_FILE="${DATA_DIR}/session.json"
if [[ ! -f "$SESSION_FILE" ]]; then
  printf '{"decision":"approve"}'
  exit 0
fi

# Parse all fields in a single python3 call. Passes path via sys.argv
# to prevent shell injection. Outputs valid JSON directly. Handles the
# TOCTOU race (file cleared between -f test and open) as "no session"
# so we don't spuriously block a completed workflow.
PARSED=$(python3 -c "
import json, sys
try:
    d = json.load(open(sys.argv[1]))
except FileNotFoundError:
    print(json.dumps({'decision': 'approve'}))
    sys.exit(0)
if d.get('status') == 'running':
    remaining = d.get('total_steps', 0) - d.get('current_index', 0)
    wf = d.get('workflow', 'unknown')
    print(json.dumps({
        'decision': 'block',
        'reason': f'Workflow {wf} incomplete — {remaining} steps remaining. Call devkit_advance to continue.'
    }))
else:
    print(json.dumps({'decision': 'approve'}))
" "$SESSION_FILE" 2>/dev/null) || {
  # Cannot parse — fail closed with diagnostic. User must either
  # complete the workflow or remove the stale file manually.
  printf 'devkit-stop-guard: cannot parse session state (python3 missing or JSON corrupt); blocking Stop\n' >&2
  printf '{"decision":"block","reason":"devkit session state corrupted — remove %s to clear"}' "$SESSION_FILE"
  exit 0
}

printf '%s' "$PARSED"
exit 0
