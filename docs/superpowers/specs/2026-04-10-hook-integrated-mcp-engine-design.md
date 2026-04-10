# Hook-Integrated MCP Engine

Redesign the devkit engine from a subprocess-spawning CLI to an MCP server + PreToolUse hook that enforces deterministic workflow execution inside Claude Code.

## Problem

The current engine spawns `claude -p` as a subprocess. Claude Code's OAuth token doesn't work for subprocess calls. The engine can't control Claude from inside Claude. All "deterministic" workflows have been running via markdown fallbacks — Claude follows them voluntarily.

## Core Insight

Claude Code has two enforcement mechanisms that can't be cheated:

1. **MCP tool scoping** — Claude can't call tools that don't exist. An MCP server that only exposes current-step tools makes skipping structurally impossible.
2. **PreToolUse hook exit 2** — hard blocks tool execution before it runs. The only native enforcement primitive in Claude Code.

Everything else (prompt instructions, CLAUDE.md, skill markdown) is advisory and degrades after ~3,000 tokens or a few turns.

## Architecture

```
Claude Code (harness)
  │
  ├── MCP Server: devkit-engine
  │     Exposes: devkit_start, devkit_advance, devkit_status
  │     + per-step dynamic tools (only current step's tools visible)
  │     State: $CLAUDE_PLUGIN_DATA/session.json + SQLite
  │
  ├── PreToolUse Hook: devkit-guard
  │     Reads session.json
  │     exit 2 blocks actions outside current step scope
  │     <2s latency budget
  │
  ├── bin/devkit (pre-compiled, auto-PATH)
  │     Used by MCP server and hook internally
  │     Also works standalone for terminal usage
  │
  └── Skills (progressive disclosure)
        Frontmatter at init (~100 tokens each)
        Full content on invoke (~500 tokens)
        Condensed principles in tool responses (~120 tokens)
```

## Enforcement Model

### Layer 1: MCP Tool Scoping (primary)

The MCP server dynamically registers tools based on workflow state. Claude can only see and call tools valid for the current step.

**No active session:**
- `devkit_start(workflow, input)` — only tool visible
- `devkit_list()` — list available workflows
- `devkit_status(session?)` — check session state

**Active session, prompt step:**
- `devkit_advance(session)` — complete current step, get next
- `devkit_status(session)` — check progress
- All standard Claude Code tools available (Bash, Read, Edit, etc.)

**Active session, command step:**
- `devkit_advance(session)` — engine runs the command internally, checks gate, advances
- Standard tools blocked by hook until command step completes

**Session complete:**
- `devkit_start` re-appears for next workflow
- `devkit_report(session)` — view results

Claude can't skip step 3 to do step 5 because step 5's advance call validates that step 4 was completed. The MCP server holds state — Claude doesn't self-report.

### Layer 2: PreToolUse Hook (safety net)

Fast guard that fires on every tool call. Reads `$CLAUDE_PLUGIN_DATA/session.json`.

```bash
# Pseudocode for devkit-guard hook
session = read_session_json()
if no session: exit 0  # no workflow active, pass through

if session.current_step.type == "command":
    if tool != "devkit_advance":
        exit 2  # block: "Complete command step first: {step.command}"

if session.enforce == "hard":
    # allow standard tools (Claude working on prompt step)
    exit 0

if session.enforce == "soft":
    # log violation, allow through
    log_violation(tool, session)
    exit 0
```

**Performance:** Read JSON file + string compare. Target <50ms. No network, no SQLite on hot path.

### Layer 3: Stop Hook (session-end gate)

Fires when Claude tries to end the session. Blocks if workflow is incomplete.

```bash
if session.status == "running":
    exit with decision: "block", reason: "Workflow {name} incomplete — {N} steps remaining"
```

## State Management

### Hot state: `$CLAUDE_PLUGIN_DATA/session.json`

```json
{
  "id": "e95e4a45bc5e",
  "workflow": "research",
  "input": "what are the best Go testing frameworks",
  "current_step": "decompose",
  "current_index": 2,
  "total_steps": 8,
  "enforce": "hard",
  "branch": false,
  "budget_usd": 0,
  "spent_usd": 0.0,
  "started_at": "2026-04-10T00:30:00Z",
  "outputs": {
    "clarify": "User wants comparison of testify vs stdlib..."
  }
}
```

Written by MCP server. Read by hook. No locking needed (single-writer, multi-reader).

### Cold state: `$CLAUDE_PLUGIN_DATA/devkit.db` (SQLite)

Session history, step records, cost tracking, reporting. Not on the hot path.

## Step Execution Flow

```
1. Claude (or user) invokes: devkit_start("research", "input")
   → MCP server: parse workflow YAML, create session.json + SQLite record
   → Returns: "Step 1 (clarify): Ask the user to sharpen the question"
   → Principles injected in response: DRY (1 line), YAGNI (1 line), etc.

2. Claude executes the step using standard tools (Bash, Read, Edit, etc.)
   → Hook allows standard tools (prompt step is active)

3. Claude calls: devkit_advance("e95e4a45bc5e")
   → MCP server: validate gate (if any), advance session.json
   → Gate passes: returns "Step 2 (decompose): Break into sub-questions"
   → Gate fails: returns error, stays on current step

4. For command steps: devkit_advance runs the command internally
   → MCP server: executes shell command, checks exit code + expect
   → Hook blocks all other tools until advance is called
   → Returns command output + next step prompt

5. Repeat until: DONE, budget exhausted, or user cancels
```

## Workflow YAML Changes

Minimal additions to existing format:

```yaml
name: self-lint
enforce: hard          # hard (default) | soft
branch: true           # create git branch (default: false)
principles: [dry, yagni, clean-code]  # auto-inject per prompt step
steps:
  - id: baseline
    command: "{input}"
    expect: success
  - id: fix
    prompt: "Fix all lint errors found above"
    principles: [clean-code]  # override per-step (optional)
    loop:
      max: 8
      gate: "{input}"
  - id: verify
    command: "{input}"
    expect: success
```

Step-level `principles` overrides workflow-level. If neither is set, implementation workflows default to `[dry, yagni, clean-code, dont-reinvent]`.

## Condensed Principles

Full skill files are 40-70 lines (~400-800 tokens). Research shows declarative lists without examples achieve ~95% compliance at ~120 tokens. The MCP server injects condensed versions:

```yaml
# skills/_principles.yml (~30 lines, ~200 tokens total)
dry:
  - Don't abstract until 3rd duplication
  - If one copy changes, must the other? If not, leave it
  - Name abstractions for what they do, not where they came from
yagni:
  - Build what's needed now, not what might be needed
  - Hardcode until configurability is actually requested
  - Premature abstraction is worse than duplication
clean-code:
  - One function, one job — if you say "and", split it
  - Names reveal intent. Booleans read as questions
  - Early returns over nesting. Max 2 indent levels
dont-reinvent:
  - stdlib > framework > established package > custom
  - Every custom solution is code you maintain forever
  - Custom justified only when existing solutions don't fit
executing:
  - One step at a time. Verify before moving on
  - Keep changes small. 20-line diff > 200-line diff
scratchpad:
  - Read .devkit/scratchpads/current.md before each iteration
  - Record what was tried and why it failed
  - 3+ failures = escalate to user
stuck:
  - Same error 2x = stop and diagnose
  - 3+ failures = escalate, don't retry
```

Injected inline in `devkit_advance` responses. Not loaded as separate skill files. Claude never reads the full skill markdown during workflow execution.

## Token Budget

Research-backed numbers for an 8-step workflow:

| Component | Tokens | When |
|-----------|--------|------|
| System prompt (cached after turn 1) | ~12,000 | Once |
| Skill frontmatter (all skills) | ~1,800 | Init |
| Per-step instruction (from advance) | ~300 | Per step |
| Condensed principles injection | ~120 | Per prompt step |
| Step output summaries passed forward | ~200 | Per handoff |
| **Total for 8-step workflow** | **~17,000** | |
| **Monolithic upfront (current)** | **~50,000+** | |

~65% token reduction vs current approach.

## Skill Auto-Activation

Skills are NOT loaded during workflow execution. The MCP server handles everything:

- **Principle skills** (dry, yagni, clean-code, dont-reinvent) → condensed rules injected by MCP per step
- **Process skills** (executing, scratchpad, stuck) → condensed rules injected by MCP for relevant step types
- **Loop detection** → scratchpad + stuck rules auto-injected when step has `loop:` field
- **Test steps** → test-gen rules injected when step ID contains "test"

Full skill files remain for explicit user invocation (`/devkit:dry`, `/devkit:clean-code`) as reference docs. They don't participate in workflow execution.

## Parallel Dispatch

For tri-review, tri-debug, tri-security — the MCP server doesn't spawn subprocesses. Instead:

```
devkit_advance returns:
  "Dispatch parallel reviews. Use the Agent tool to launch:
   1. reviewer agent with this prompt: {prompt}
   2. /codex:rescue with this prompt: {prompt}
   3. /gemini:rescue with this prompt: {prompt}
   Consolidate results and call devkit_advance when done."
```

Claude Code dispatches using its native Agent tool and plugins. The engine tracks which parallel steps completed via `devkit_advance` calls.

## Git Branch Behavior

Per-workflow opt-in via `branch: true` in YAML:

| Workflow type | Default | Reason |
|---------------|---------|--------|
| research, deep-research, tri-* | `branch: false` | Read-only, no file changes |
| self-lint, self-test, autoloop | `branch: true` | Modifies code, needs revert safety |
| feature, bugfix, refactor | `branch: false` | User is already on a feature branch |

When `branch: true`: engine creates branch on `devkit_start`, reverts on gate failure, commits on success.

## Binary Distribution

Research finding: `bin/` directory contents are auto-added to PATH by Claude Code plugin system.

- Ship pre-compiled binaries in `bin/` for each platform
- Release workflow builds 6 targets (linux/darwin/windows x amd64/arm64)
- `scripts/install-engine.sh` remains for manual/terminal installation
- Inside Claude Code: binary is just there, no bootstrap needed

## Backward Compatibility

- **Terminal usage:** `devkit workflow run research "input"` still works with subprocess runners (Codex, Gemini). The `--agent` flag selects runner.
- **Claude Code usage:** MCP server mode auto-detected. No subprocess runners needed.
- **Existing YAML workflows:** All valid. New fields (`enforce`, `branch`, `principles`) are optional with sensible defaults.
- **Existing hooks:** Unchanged. `devkit-guard` is a new hook added alongside existing ones.

## What Changes vs Current

| Component | Current | New |
|-----------|---------|-----|
| Engine role | CLI that spawns subprocesses | MCP server + state machine |
| Claude runner | `claude -p` (broken) | Removed — Claude Code IS the runner |
| Enforcement | None (markdown honor system) | MCP tool scoping + PreToolUse exit 2 |
| Principle skills | Loaded if Claude decides to | Injected by engine per step |
| Token usage | ~50k+ for 8-step workflow | ~17k |
| State | SQLite only | session.json (hot) + SQLite (cold) |
| Binary distribution | Install script | `bin/` auto-PATH + install script fallback |

## What Stays

- All 18 YAML workflow definitions
- SQLite session/step history and reporting
- Budget enforcement
- Gate/loop/branch logic
- Codex/Gemini subprocess runners for terminal
- All existing hooks
- Skill files as reference docs

## Implementation Scope

1. **MCP server** — Go binary that speaks MCP protocol, manages session state, parses workflows, injects principles
2. **PreToolUse hook** — shell script, reads session.json, exit 2 on violations
3. **Stop hook** — blocks session end if workflow incomplete
4. **`_principles.yml`** — condensed principle index
5. **Workflow YAML updates** — add `enforce`, `branch`, `principles` fields with defaults
6. **Plugin manifest** — register MCP server
7. **Binary distribution** — ship in `bin/`, update release workflow
