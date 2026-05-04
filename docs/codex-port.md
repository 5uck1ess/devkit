# Codex Portability Notes

devkit has a shared deterministic core and host-specific enforcement adapters.

## Portable Core

These parts are host-neutral and should be shared by Claude Code, Codex, and other MCP clients:

- YAML workflow schema and validation.
- MCP tools: `devkit_start`, `devkit_advance`, `devkit_status`, `devkit_list`.
- Session state in `session.json`.
- SQLite history and reports.
- Engine-owned command steps.
- Branch, loop, gate, and output interpolation behavior.
- CLI runners for Claude, Codex, Gemini, and local OpenAI-compatible endpoints.

## Host Hook Enforcement

The hard guardrail stack started on Claude Code lifecycle hooks:

- `PreToolUse`: safety checks, audit trail, PR gate, security pattern checks, and workflow tool gating.
- `PostToolUse`: output/write validation, slop detection, and language review.
- `SubagentStop`: subagent completion checks.
- `Stop`: quality gate and incomplete-workflow stop guard.

Codex now exposes an official hook lifecycle with `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PermissionRequest`, `PostToolUse`, and `Stop`. The repo-local `.codex/config.toml` enables hooks and `.codex/hooks.json` ports the Claude stack where Codex can observe the event:

- `PreToolUse`: Bash safety/audit/PR checks, file safety/security checks for `apply_patch`, and workflow gating.
- `PermissionRequest`: approval-time hard-deny safety checks.
- `PostToolUse`: Bash output validation and file validation/review for `apply_patch`.
- `Stop`: quality gate and incomplete-workflow continuation guard.

Known gaps remain. Codex `PreToolUse` and `PostToolUse` currently cover Bash, `apply_patch`, and MCP calls, but interception is incomplete for richer shell paths such as `unified_exec` and does not cover non-shell/non-MCP tools such as web search. Codex also has no direct `SubagentStop` equivalent.

## Recommended Codex Scope

Start with an additive adapter:

1. Register `bin/devkit mcp` in Codex config.
2. Enable `[features].codex_hooks = true`.
3. Trust the project `.codex/` layer so `.codex/config.toml` and `.codex/hooks.json` load.
4. Load `AGENTS.md` as project guidance.
5. Use shared workflows and skills with Codex-aware dispatch wording.
6. Keep Claude packaging untouched.

Then harden where it matters:

1. Prefer workflow command/gate steps for actions that must be mechanically controlled.
2. Add `devkit_advance` validation for required output shapes and sentinels using `require:`.
3. Use CI or git hooks for post-facto quality checks.
4. Build a Codex shim only if the remaining hook interception gaps matter for the workflow.

## Output Contracts

Workflow steps can declare a small host-neutral `require:` block. The MCP server and terminal runner both validate it before the workflow can advance.

Supported checks:

- `non_empty: true`
- `contains: ["literal text"]`
- `until: "SENTINEL"` using the same word-boundary matching as loop `until`
- `last_line_regex: "^PR: ([0-9]+|FAILED .+)$"`

Use this for values that later steps parse, not as a substitute for tests or review. For example, `pr-ready.create-pr` requires the last output line to be either `PR: <number>` or `PR: FAILED <reason>` so the monitor step never starts from malformed PR state.

## Compatibility Matrix

| Capability | Claude Code | Codex Baseline | Codex With Future Shim |
|---|---:|---:|---:|
| MCP workflow state | Full | Full | Full |
| YAML step order | Full | Full | Full |
| Engine command steps | Full | Full | Full |
| Pre-tool blocking | Full | Partial for Bash/apply_patch/MCP | Full remaining-gap coverage |
| Permission request control | N/A | Partial deny/allow support | Full remaining-gap coverage |
| Post-tool validation | Full | Partial for Bash/apply_patch/MCP | Full remaining-gap coverage |
| Stop blocking | Full | Continuation via `Stop` hook | Full remaining-gap coverage |
| Subagent stop validation | Full | Advisory | Possible |
| Plugin/bundle packaging | `.mcpb` | config.toml + `.codex/config.toml` + `.codex/hooks.json` | config.toml + hooks + shim |
