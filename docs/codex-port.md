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

## Claude-Only Hard Enforcement

The current hard guardrail stack depends on Claude Code lifecycle hooks:

- `PreToolUse`: safety checks, audit trail, PR gate, security pattern checks, and workflow tool gating.
- `PostToolUse`: output/write validation, slop detection, and language review.
- `SubagentStop`: subagent completion checks.
- `Stop`: quality gate and incomplete-workflow stop guard.

Codex does not expose the same hook lifecycle to this repo. Under Codex, those checks become advisory unless they are moved into the engine, CI, git hooks, or an external Codex shim.

## Recommended Codex Scope

Start with an additive adapter:

1. Register `bin/devkit mcp` in Codex config.
2. Load `AGENTS.md` as project guidance.
3. Use shared workflows and skills with Codex-aware dispatch wording.
4. Keep Claude packaging untouched.

Then harden where it matters:

1. Prefer workflow command/gate steps for actions that must be mechanically controlled.
2. Add `devkit_advance` validation for required output shapes and sentinels.
3. Use CI or git hooks for post-facto quality checks.
4. Build a Codex shim only if pre-tool blocking is required.

## Compatibility Matrix

| Capability | Claude Code | Codex Baseline | Codex With Future Shim |
|---|---:|---:|---:|
| MCP workflow state | Full | Full | Full |
| YAML step order | Full | Full | Full |
| Engine command steps | Full | Full | Full |
| Pre-tool blocking | Full | Advisory | Possible |
| Post-tool validation | Full | Manual/CI | Possible |
| Stop blocking | Full | Advisory | Possible |
| Subagent stop validation | Full | Advisory | Possible |
| Plugin/bundle packaging | `.mcpb` | config.toml | config.toml + shim |
