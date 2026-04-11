# CLAUDE.md

Navigation map for Claude Code working in this repo. Anchor-style, not narrative — keep it short. `pr-ready`'s doc-check targets this file, so stale entries get caught at PR time.

devkit is a Claude Code plugin: deterministic YAML workflow engine, thin-dispatcher skills, enforcement hooks, multi-agent consensus.

## Layout

| Path | What | Grep here for |
|---|---|---|
| `skills/` | 38 SKILL.md dispatchers | Skill descriptions; what triggers each workflow |
| `skills/_principles.yml` | Shared cross-cutting principle config | Rules applied to every skill |
| `skills/creating-workflows/` | Workflow YAML schema reference | Step types, `parallel:`/`branch:`/`loop:`/`expect:` semantics |
| `workflows/` | 21 YAML workflow definitions | What each skill actually runs |
| `src/engine/engine.go` | Go workflow executor | How `parallel: [ids]` skips-then-fans-out (`parallelChildren`), branch eval, loop gates |
| `src/engine/workflow.go` | Workflow struct + YAML parsing | Step field definitions |
| `src/cmd/guard.go` | Hook gatekeeper | What Bash/Edit/etc. is allowed per step type + enforce level |
| `src/cmd/workflow.go` | `devkit workflow` CLI command | Workflow invocation entry point |
| `src/cmd/mcp.go` | `devkit mcp` CLI command | MCP server bootstrap |
| `src/mcp/tools.go` | MCP tool schemas | `devkit_start`, `devkit_advance`, `devkit_list`, `devkit_status` contracts |
| `src/mcp/server.go` | MCP server implementation | Tool dispatching |
| `src/mcp/principles.go` | Principle injection | How `_principles.yml` reaches workflows |
| `src/runners/runner.go` | Model tier interface | The `smart`/`general`/`fast` contract |
| `src/runners/{claude,codex,gemini}.go` | Tier implementations | How each external CLI is called |
| `src/lib/state.go` + `state_json.go` | Workflow state persistence | Running-state schema, step outputs |
| `src/lib/state_lock_{unix,windows}.go` | Cross-platform file locking | OS-specific state lock |
| `src/lib/git.go` | Git helpers | Diff collection, branch checks |
| `src/lib/report.go` | Final-report formatting | Workflow output rendering |
| `hooks/hooks.json` | PreToolUse/PostToolUse/SubagentStop/Stop wiring | Which shell script runs for which event |
| `hooks/*.sh` | Individual hook scripts | safety-check, audit-trail, pr-gate, stop-gate, lang-review, etc. |
| `agents/*.md` | 6 subagent definitions | documenter, improver, researcher, reviewer, security-auditor, test-writer |
| `mcpb/` | MCPB bundle (launcher, manifest.json, server) | Packaged distribution artifact |
| `bin/devkit` | User-facing CLI wrapper | Shells out to the `devkit-engine` Go binary |
| `resources/rules/` | Language coding rules | Installed via the `setup-rules` skill |
| `.claude-plugin/plugin.json` | Plugin manifest | Name, version, `mcpServers` pointer |
| `src/Makefile` | Build + test + version sync | `make build`, `make test`, `make check`, `make sync-version` |
| `commands/references/` | 3 reference files pulled in by skills (`debug-checklists.md`, `domain-probes.md`, `stub-patterns.md`) | Shared checklist/probe/stub content; write new work as skills |

## Architectural invariants

- **Workflows are deterministic.** The engine controls step sequencing, loops, branches, and gates — not the model. Skills are thin dispatchers that call `devkit_start` then loop on `devkit_advance` until done.
- **Parallel step pattern.** Steps listed in another step's `parallel: [ids]` are skipped during the sequential walk (see `parallelChildren` in `src/engine/engine.go`) and dispatched concurrently via `runParallel`. Never expect a step that appears in a parallel list to also execute in its sequential position.
- **Enforce levels gate hook permissions.** `enforce: soft` lets gather/setup steps run `git diff` and other shell work during dispatch; `enforce: hard` blocks them. The gatekeeper logic lives in `src/cmd/guard.go`; see `guard_test.go` for the allow/deny matrix.
- **Engine binary is separate from the CLI wrapper.** `bin/devkit` is the committed shell wrapper; it execs `devkit-engine` (compiled Go, gitignored). The wrapper handles install/version checks; the engine handles workflow execution.
- **Skills are the future; `commands/` is legacy.** PR #77 migrated commands to skills. Both auto-discover, but skills support paths, disable-model-invocation, and supporting files.

## Conventions

- Never push directly to `main`. Always PR. Direct push bypasses the version-bump and release pipeline.
- Never amend commits — create new ones, even after pre-commit hook failures.
- Never skip hooks (`--no-verify`, `--no-gpg-sign`) without explicit user consent.
- Commit message style: check recent `git log` before writing one. Repo uses conventional-commit prefixes (`docs:`, `fix:`, `feat:`, `refactor:`, `chore:`).
- Version bumps happen automatically on PR merge — don't hand-edit `.claude-plugin/plugin.json` version.
- `CHANGELOG.md` is managed by the release pipeline, not by `pr-ready`'s doc-check or manual edits.
- No direct edits to `~/Documents/LocalDev/claude-shared/` — the OneDrive copy is primary; the local path is a GitHub backup.

## Fixing this file

If a row above is wrong or a lookup keeps failing, fix it in the same PR that changed the underlying code — `pr-ready`'s doc-check targets `CLAUDE.md`. If you find a recurring exploration query that isn't covered, add a one-liner anchor to the layout table. Keep it short; this file exists to save tokens, not to explain devkit.
