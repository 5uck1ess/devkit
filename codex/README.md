# Codex Adapter

This directory contains the Codex-facing adapter for devkit. It is additive: Claude Code packaging remains in `.claude-plugin/`, `mcpb/`, and `hooks/`.

## What Works

- MCP workflow tools: `devkit_start`, `devkit_advance`, `devkit_status`, `devkit_list`.
- YAML workflow parsing, step ordering, branches, loops, gates, session state, and reports.
- Engine-owned command steps.
- Codex CLI runner for terminal workflow execution.

## Enforcement Difference

Claude Code supports lifecycle hooks that devkit uses for hard guardrails. Codex does not provide equivalent repo/plugin hooks for `PreToolUse`, `PostToolUse`, `Stop`, or `SubagentStop`.

That means Codex support is split into two levels:

- Baseline Codex adapter: deterministic MCP workflow state, advisory tool discipline, Codex sandbox/approval controls.
- Future shim: an external launcher/proxy that reuses devkit guard policy to intercept actions before Codex performs them.

## Install Sketch

Build or install the devkit binary, then register it as an MCP server in Codex config. See `config.example.toml` for a template.

The server command is:

```bash
bin/devkit mcp
```

Use an absolute path in user config so Codex can launch it from any workspace.
