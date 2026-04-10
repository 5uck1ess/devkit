#!/usr/bin/env bash
set -euo pipefail

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

STATUS=$(python3 -c "import json; d=json.load(open('$SESSION_FILE')); print(d.get('status',''))" 2>/dev/null || echo "")
WORKFLOW=$(python3 -c "import json; d=json.load(open('$SESSION_FILE')); print(d.get('workflow',''))" 2>/dev/null || echo "")
CURRENT=$(python3 -c "import json; d=json.load(open('$SESSION_FILE')); print(d.get('current_index',0))" 2>/dev/null || echo "0")
TOTAL=$(python3 -c "import json; d=json.load(open('$SESSION_FILE')); print(d.get('total_steps',0))" 2>/dev/null || echo "0")

if [[ "$STATUS" == "running" ]]; then
  REMAINING=$((TOTAL - CURRENT))
  printf '{"decision":"block","reason":"Workflow %s incomplete — %d steps remaining. Call devkit_advance to continue."}' "$WORKFLOW" "$REMAINING"
  exit 0
fi

printf '{"decision":"approve"}'
exit 0
