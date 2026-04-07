---
description: Multi-agent debugging — send a bug report to available agents (Claude + Codex + Gemini) via plugin or CLI, get independent root-cause hypotheses, and a consensus fix.
---

# Triple-Agent Debug

Send a bug description to all available agents in parallel, get independent root-cause analyses, and consolidate into a recommended fix.

## Step 0: Harness Detection

```bash
if command -v devkit >/dev/null 2>&1; then
  echo "Go harness detected — delegating to devkit dispatch for full output capture."
  devkit dispatch {prompt with bug context}
  exit 0
fi
```

If the `devkit` binary is in PATH, delegate entirely to it. Only fall through to plugin-based steps if the harness is not installed.

## Step 1: Gather Context

Collect from the user:
- Bug description or error message
- Stack trace (if available)
- Steps to reproduce (if known)
- Relevant files

If a stack trace is provided, extract the relevant source files:
```bash
# Parse file paths from stack trace and read them
```

## Step 2: Detect Available Agents

Check for plugins first (preferred), then fall back to CLI:

```bash
# Plugin detection (preferred — structured job management)
HAS_CODEX_PLUGIN=$(/codex:status >/dev/null 2>&1 && echo "yes" || echo "no")
HAS_GEMINI_PLUGIN=$(/gemini:status >/dev/null 2>&1 && echo "yes" || echo "no")

# CLI fallback detection
HAS_CODEX_CLI=$(command -v codex && echo "yes" || echo "no")
HAS_GEMINI_CLI=$(command -v gemini && echo "yes" || echo "no")
```

Run with whatever is available. 1 agent minimum (Claude), up to 3. Prefer plugin over CLI.

## Step 3: Build the Prompt

```
Debug this issue. Provide:
1. Root cause — what is actually wrong and why
2. Evidence — specific lines of code that cause the bug
3. Fix — exact code changes to resolve it
4. Verification — how to confirm the fix works

Bug: {description}
Stack trace: {stack_trace}
Relevant code: {source_files}
```

## Concurrency & Budget

- **Concurrency limit:** Max 3 parallel agents.
- **Token budget:** ~300k tokens across all agents.
- **Rate limiting:** If API throttles, stagger agent launches.

## Step 4: Dispatch (Hybrid, Graceful Degradation)

**[PARALLEL]** Launch all available agents concurrently:

**CRITICAL:** All context (bug description, stack trace, relevant code) MUST be passed inline in each agent's prompt. Worktree-isolated agents cannot see the latest commits or local state.

### Claude — always runs (native background agent)

```
Task: Debug this issue.
Agent: researcher
Input: {prompt}

{context — bug description, stack trace, relevant code inlined here}
```

<!-- The orchestrator MUST inline all context here. The agent runs in a worktree and cannot fetch it. -->

### Codex — if available

```
/codex:rescue --effort high --background "{prompt} {context}"
```

Retrieve result with `/codex:result` when done. Omit `--model` to use the account default.

### Gemini — if available

**Plugin (preferred):**

```
/gemini:rescue --background "{prompt} {context}"
```

Retrieve result with `/gemini:result` when done. Omit `--model` to use the account default.

**CLI fallback (only if plugin not installed):**

```bash
if [ "$HAS_GEMINI_CLI" = "yes" ]; then
  gemini -p "{prompt} {context}" -y \
    --output-format text > /tmp/tri-debug-gemini.txt 2>&1 &
  GEMINI_PID=$!
fi

wait
```

## Step 5: Consolidate

```
## Debug Report: {summary}

### Agents Used: {count}/3
{list which agents ran}

### Consensus Root Cause (agreed by {n}+ agents)
- ...

### Root Cause Analysis by Agent
- **Claude:** ...
- **Codex:** ... (if available)
- **Gemini:** ... (if available)

### Recommended Fix
{merged fix based on consensus}

### Verification Steps
1. ...
```

## Investigation Techniques

When building the debug prompt or analyzing results, apply the most appropriate technique:

| Technique | When to Use | How |
|-----------|-------------|-----|
| **Binary search** | Bug exists somewhere in a range of changes/code | Bisect the search space — disable half, test, narrow |
| **Differential debugging** | "It worked before" | Compare working vs broken state — `git diff`, env diff, config diff |
| **Minimal reproduction** | Complex bug with many variables | Strip away everything unrelated until the simplest trigger remains |
| **Trace execution** | Control flow is unclear | Add logging or step through — follow the actual path, not the assumed one |
| **Working backwards** | You know the symptom but not the cause | Start at the error, trace data/control flow backwards to the source |
| **5 Whys** | Surface fix isn't enough | Ask "why?" at each layer: symptom → immediate cause → deeper cause → root cause → systemic issue |

## Domain-Specific Debugging Checklists

When the bug domain is identifiable (API, database, auth, async, performance), read the relevant section from `commands/references/debug-checklists.md` and include it in each agent's prompt. Only include the matching domain — don't load the full file.

## Rules

- Claude always runs — Codex and Gemini are optional
- If only Claude is available, still provide the full report format (just one perspective)
- Report which agents participated
- If agents disagree on root cause, present all hypotheses ranked by evidence
- When agents agree on symptoms but disagree on root cause, apply the 5 Whys to the shared symptoms to converge on a deeper cause. If they disagree on symptoms too, prioritize verifying symptoms through additional logs or traces before root-cause analysis
- Include the relevant domain checklist in each agent's prompt when the bug category is identifiable
- Clean up temp files after
