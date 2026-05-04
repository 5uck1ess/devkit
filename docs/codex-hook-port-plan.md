# Codex Hook Port Plan

Goal: port devkit's Claude hook enforcement to Codex using official Codex hooks, while keeping Claude packaging untouched.

## Current State

Branch: `codex-adaptability-review`

Committed baseline:

- `98bddc3 Add Codex adapter baseline`
- `c807c6f Add workflow output requirements`

Official docs: <https://developers.openai.com/codex/hooks>

Key facts from the docs:

- Codex supports `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PermissionRequest`, `PostToolUse`, and `Stop`.
- Hooks can live in `~/.codex/hooks.json`, `~/.codex/config.toml`, `<repo>/.codex/hooks.json`, or `<repo>/.codex/config.toml`.
- Installed Codex plugins can bundle lifecycle config through a plugin manifest or default `hooks/hooks.json`.
- `PreToolUse` can intercept `Bash`, `apply_patch`, and MCP tool calls.
- `PostToolUse` runs after supported tools, including `Bash`, `apply_patch`, and MCP calls.
- `Stop` can continue the turn by returning a block/continuation reason.
- `PermissionRequest` can allow or deny approval requests.
- Limitation: `PreToolUse` and `PostToolUse` do not intercept every shell path yet, especially richer `unified_exec` cases, and do not cover tools like `WebSearch`.

## 1. Update Assumptions

- Revise `codex/README.md`, `docs/codex-port.md`, and `README.md`.
- Replace "Codex has no equivalent hooks" with the accurate version: Codex supports hooks, but interception is incomplete for some shell paths and non-shell tools.
- Document supported events: `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PermissionRequest`, `PostToolUse`, `Stop`.

## 2. Add Codex Hook Config

- Add `<repo>/.codex/hooks.json`.
- Mirror the current Claude `hooks/hooks.json` where possible.
- Map Codex event names:
  - `PreToolUse`: safety, audit, PR gate, security patterns, devkit guard.
  - `PostToolUse`: post-validate, slop-detect, lang-review.
  - `Stop`: stop-gate, devkit-stop-guard.
  - `PermissionRequest`: approval-specific safety guard.
- Skip `SubagentStop` initially unless Codex adds an equivalent subagent event.

## 3. Add Payload Compatibility Layer

- Current scripts expect Claude hook payloads and/or Claude plugin env vars.
- Add a shared parser script, likely `hooks/lib/read-codex-hook.sh` or equivalent.
- Normalize Codex fields into the same env vars existing hooks use:
  - `HOOK_EVENT_NAME`
  - `TOOL_NAME`
  - `TOOL_INPUT_COMMAND`
  - `TOOL_RESPONSE`
  - `CWD`
  - `SESSION_ID`
  - `TURN_ID`
- Preserve Claude compatibility by detecting payload shape rather than replacing existing behavior.

## 4. Port `devkit-guard` First

- Start with `PreToolUse` for `Bash`, `apply_patch`, `Edit`, `Write`, and MCP tool names.
- Confirm it can block:
  - bad Bash command
  - out-of-step Bash during hard prompt step
  - write/edit during hard prompt step
  - command-step tool misuse
- Use Codex-supported block output:
  - preferred: `hookSpecificOutput.permissionDecision = "deny"`
  - fallback: exit code `2` with stderr reason

## 5. Port Stop Guard

- Wire `Stop` to `devkit-stop-guard`.
- Codex `Stop` block means "continue with this reason", not exactly "reject turn".
- Adapt output JSON accordingly.
- Test active workflow blocks stopping and prompts Codex to call `devkit_advance`.

## 6. Port Safety And Post Hooks

Port in this order:

1. `safety-check.sh`
2. `security-patterns.sh`
3. `post-validate.sh`
4. `audit-trail.sh`
5. `pr-gate.sh`
6. `slop-detect.sh`
7. `lang-review.sh`
8. `stop-gate.sh`

Leave `rtk-rewrite.sh` for later unless Codex supports input rewriting. Current docs say `updatedInput` is parsed but unsupported/fail-open for `PreToolUse`.

## 7. Add Hook Tests

- Add a local test harness under `codex/hooks_test/` or extend `hooks/hooks_test.sh`.
- Test scripts by piping representative Codex hook payloads.
- Include fixtures for:
  - `PreToolUse` Bash
  - `PreToolUse apply_patch`
  - `PostToolUse` Bash
  - `PostToolUse apply_patch`
  - `Stop`
  - MCP tool call
- Run existing `hooks/hooks_test.sh` to ensure Claude behavior did not regress.

## 8. End-To-End Codex Probe

Use a temporary test workflow/session.

Confirm:

- Project-local `.codex/hooks.json` loads.
- `PreToolUse` fires for simple Bash.
- `PreToolUse` blocks with exit `2` or deny JSON.
- `PostToolUse` fires after Bash.
- `Stop` continues the turn when workflow is incomplete.
- `apply_patch` hook fires, or document limitations if not observable from CLI.

## 9. Update Compatibility Matrix

Update matrix from:

- `Pre-tool blocking: Advisory`

To:

- `Pre-tool blocking: Partial, supported for Bash/apply_patch/MCP; incomplete for some shell paths and non-shell tools`

Update `Codex With Future Shim` to cover only remaining gaps, not all hooks.

## 10. Suggested Commit Scope

Recommended commits:

1. Docs update for official Codex hooks.
2. `.codex/hooks.json` plus payload normalization.
3. `devkit-guard` and stop guard port.
4. Safety/post hook ports plus tests.
