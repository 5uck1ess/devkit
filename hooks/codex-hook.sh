#!/usr/bin/env bash
set -euo pipefail

# Codex hook adapter for the existing Claude-oriented hook scripts.
#
# Codex and Claude use the same top-level hook fields for Bash and MCP
# calls, but Codex reports file edits as tool_name="apply_patch" with
# the patch text in tool_input.command. Existing devkit hooks expect
# Edit/Write-style payloads with file_path/content. Normalize only that
# shape and then delegate to the requested hook unchanged.

if [[ $# -lt 1 ]]; then
  printf 'codex-hook: usage: codex-hook.sh <hook-script>\n' >&2
  exit 2
fi

target="$1"
shift || true

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

case "$target" in
  /*) target_path="$target" ;;
  *) target_path="$script_dir/$target" ;;
esac

if [[ ! -x "$target_path" ]]; then
  printf 'codex-hook: target is not executable: %s\n' "$target_path" >&2
  exit 2
fi

export CLAUDE_PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$repo_root}"
export CLAUDE_PLUGIN_DATA="${CLAUDE_PLUGIN_DATA:-$repo_root/.devkit}"

input="$(cat)"

if [[ -z "$input" ]]; then
  printf '%s' "$input" | "$target_path" "$@"
  exit $?
fi

tool_name="$(printf '%s' "$input" | jq -r '.tool_name // empty' 2>/dev/null || true)"

if [[ "$tool_name" == "apply_patch" ]]; then
  input="$(printf '%s' "$input" | jq '
    def patch_path:
      (.tool_input.command // "")
      | split("\n")
      | map(capture("^\\*\\*\\* (Add|Update|Delete) File: (?<path>.+)$")? | .path)
      | map(select(. != null and . != ""))
      | .[0] // "";

    .tool_name = "Write"
    | .tool_input.file_path = patch_path
    | .tool_input.content = (.tool_input.command // "")
  ' 2>/dev/null || printf '%s' "$input")"
fi

printf '%s' "$input" | "$target_path" "$@"
