# Codex Adapter

This directory contains the Codex-facing adapter for devkit. It is additive: Claude Code packaging remains in `.claude-plugin/`, `mcpb/`, and `hooks/`.

## What Works

- MCP workflow tools: `devkit_start`, `devkit_advance`, `devkit_status`, `devkit_list`.
- YAML workflow parsing, step ordering, branches, loops, gates, session state, and reports.
- Engine-owned command steps.
- Workflow `require:` output contracts checked at `devkit_advance`.
- Codex CLI runner for terminal workflow execution.
- Codex lifecycle hooks for Bash, `apply_patch`, MCP calls, approval requests, post-tool checks, and Stop continuation when `codex_hooks` is enabled.

## Enforcement Difference

Claude Code and Codex both support lifecycle hooks, but the coverage and contracts are not identical. Codex currently supports `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PermissionRequest`, `PostToolUse`, and `Stop`.

That means Codex support is split into two levels:

- Baseline Codex adapter: deterministic MCP workflow state, engine-owned command steps, Codex sandbox/approval controls, and partial hook enforcement through `.codex/hooks.json`.
- Remaining gaps: Codex hooks do not intercept every shell path yet, especially richer `unified_exec` cases, and do not cover non-shell/non-MCP tools such as web search. `SubagentStop` has no direct Codex equivalent.
- Future shim: an external launcher/proxy only for the remaining interception gaps if full host-level enforcement is required.

## Install Sketch

Build or install the devkit binary, enable Codex hooks, then register devkit as an MCP server in Codex config. See `config.example.toml` for a template.

The server command is:

```bash
bin/devkit mcp
```

Use an absolute path in user config so Codex can launch it from any workspace. Project-local hook config lives in `.codex/config.toml` and `.codex/hooks.json`; Codex loads it only when the project `.codex/` layer is trusted.
