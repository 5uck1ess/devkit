#!/usr/bin/env bash
set -euo pipefail

# devkit-stop-guard: Stop hook that blocks session end during active workflows.
# Outputs JSON: {"decision":"approve"} or {"decision":"block","reason":"..."}.

DATA_DIR="${CLAUDE_PLUGIN_DATA:-}"
if [[ -z "$DATA_DIR" ]]; then
  printf '{"decision":"approve"}'
  exit 0
fi

SESSION_FILE="${DATA_DIR}/session.json"
if [[ ! -f "$SESSION_FILE" ]]; then
  printf '{"decision":"approve"}'
  exit 0
fi

# Parse all fields in a single python3 call. Passes path via sys.argv
# to prevent shell injection. Outputs valid JSON directly.
python3 -c "
import json, sys
d = json.load(open(sys.argv[1]))
if d.get('status') == 'running':
    remaining = d.get('total_steps', 0) - d.get('current_index', 0)
    wf = d.get('workflow', 'unknown')
    print(json.dumps({
        'decision': 'block',
        'reason': f'Workflow {wf} incomplete — {remaining} steps remaining. Call devkit_advance to continue.'
    }))
else:
    print(json.dumps({'decision': 'approve'}))
" "$SESSION_FILE" 2>/dev/null || {
  # Cannot parse — approve to avoid trapping the user
  printf '{"decision":"approve"}'
}

exit 0
