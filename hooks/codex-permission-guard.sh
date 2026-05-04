#!/usr/bin/env bash
set -euo pipefail

# PermissionRequest adapter. Codex approval hooks use a different JSON
# decision shape than PreToolUse. Reuse safety-check's hard-deny rules:
# exit 2 becomes a PermissionRequest deny; ask/allow cases fall through
# to the normal Codex approval prompt.

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

export CLAUDE_PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$repo_root}"
export CLAUDE_PLUGIN_DATA="${CLAUDE_PLUGIN_DATA:-$repo_root/.devkit}"

input="$(cat)"
tmp_err="$(mktemp)"
trap 'rm -f "$tmp_err"' EXIT

set +e
printf '%s' "$input" | "$script_dir/codex-hook.sh" safety-check.sh >/dev/null 2>"$tmp_err"
rc=$?
set -e

if [[ "$rc" -eq 2 ]]; then
  reason="$(tr '\n' ' ' < "$tmp_err" | sed 's/[[:space:]]*$//')"
  [[ -n "$reason" ]] || reason="Blocked by devkit safety policy."
  jq -n --arg reason "$reason" '{
    hookSpecificOutput: {
      hookEventName: "PermissionRequest",
      decision: {
        behavior: "deny",
        message: $reason
      }
    }
  }'
  exit 0
fi

exit 0
