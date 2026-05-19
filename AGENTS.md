# Devkit For Codex

devkit is a deterministic workflow engine. In this repo, treat the YAML workflow and MCP state as the source of truth: start a workflow, execute only the current step, report the step result, then advance.

## Workflow Discipline

- Use `devkit_start` to begin a workflow and `devkit_advance` to move to the next step.
- Do not skip, reorder, or merge workflow steps unless the engine returns a branch/loop transition that does so.
- For command steps, do not run the command yourself. Call `devkit_advance`; the engine owns command execution.
- For prompt steps, do the requested work, summarize the result as the step output, then call `devkit_advance`.
- For loop steps, keep the iteration small and use `.devkit/scratchpads/current.md` when the step asks for it.
- If an active workflow exists, finish or explicitly stop it before starting a different workflow.

## Tool Mapping

- Claude `Bash` maps to Codex shell execution.
- Claude `Read`, `Grep`, and `Glob` map to shell reads, `rg`, and file inspection.
- Claude `Edit` and `Write` map to `apply_patch` for manual edits.
- Claude `Agent` and `Task` map to Codex subagents when the current environment exposes them; otherwise use external CLI runners or stop and report that model-diverse dispatch is unavailable.
- Claude `WebFetch` and `WebSearch` map to Codex web tools only when browsing is available and allowed.

## Enforcement Notes

Claude Code installs lifecycle hooks from `hooks/hooks.json` and can hard-block out-of-step tools. Codex-specific lifecycle hooks, prompt bridges, and config are planned for the separate `devkit-codex` adapter, which points Codex at this repo's MCP engine once released. Follow the same workflow policy voluntarily here, and rely on Codex sandbox/approval boundaries plus the MCP engine for stateful workflow control.

Hard guarantees that still hold under Codex:

- The MCP engine controls step order.
- Command steps execute inside the engine.
- Session state, branches, loops, gates, and reports remain engine-owned.

Guarantees that are adapter-owned under Codex:

- Blocking arbitrary shell/edit tool use during hard prompt steps.
- Running post-tool validation automatically after every edit or command.
- Blocking assistant stop while a workflow is incomplete.
- Validating subagent completion through a host lifecycle event.

## Repo Boundaries

- Keep Claude packaging in `.claude-plugin/`, `mcpb/`, and `hooks/` working.
- Keep Codex-specific integration in the separate `devkit-codex` repository once it is released, unless the change is shared engine behavior.
- Do not hand-edit generated binaries or bundled release artifacts.
