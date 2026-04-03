---
name: tri:debug
description: Multi-agent debugging — send a bug report to available agents (Claude + Codex + Gemini) via plugin or CLI, get independent root-cause hypotheses, and a consensus fix.
---

# Triple-Agent Debug

Send a bug description to all available agents in parallel, get independent root-cause analyses, and consolidate into a recommended fix.

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

### Claude — always runs (native background agent)

```
Task: Debug this issue using the researcher agent.
Agent: researcher
Input: {prompt} + {context}
```

### Codex — if available

```
/codex:rescue --model gpt-5.4 --effort high --background "{prompt}"
```

Retrieve result with `/codex:result` when done.

### Gemini — if available

**Plugin (preferred):**

```
/gemini:rescue --model gemini-3.1-pro --background "{prompt}"
```

Retrieve result with `/gemini:result` when done.

**CLI fallback (only if plugin not installed):**

```bash
if [ "$HAS_GEMINI_CLI" = "yes" ]; then
  gemini -p "{prompt}" -m gemini-3.1-pro -y \
    --output-format text > /tmp/tri-debug-gemini.txt 2>/dev/null &
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

## Rules

- Claude always runs — Codex and Gemini are optional
- If only Claude is available, still provide the full report format (just one perspective)
- Report which agents participated
- If agents disagree on root cause, present all hypotheses ranked by evidence
- Clean up temp files after
