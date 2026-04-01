---
name: tri:review
description: Triple-agent PR/code review. Claude runs as native background agent (token-efficient), Codex and Gemini via CLI. Consolidates findings.
---

# Triple-Agent Review

Run the same code review across three AI agents in parallel and consolidate results.

## Step 1: Gather Context

```bash
git diff main...HEAD > /tmp/tri-review-diff.txt
```

If empty, try `git diff --cached` or ask what to review.

## Step 2: Build the Prompt

Use the user's custom prompt if provided. Otherwise default to:

```
Review this code diff. For each issue found, report:
- File and line number
- Severity (critical / warning / suggestion)
- Description of the issue
- Suggested fix

Focus on: bugs, security issues, DRY violations, unnecessary complexity, missing edge cases.
```

## Step 3: Detect Available Agents

```bash
HAS_CODEX=$(command -v codex && echo "yes" || echo "no")
HAS_GEMINI=$(command -v gemini && echo "yes" || echo "no")
```

Run with whatever is available. Claude always runs. Codex and Gemini are optional.

## Step 4: Dispatch in Parallel (Hybrid, Graceful Degradation)

### Claude — always runs (native background agent, token-efficient)

Spawn the `reviewer` agent as a background task. The orchestrator only receives the summary, not the full conversation.

```
Task: Review this diff using the reviewer agent.
Agent: reviewer
Input: {prompt} + {diff}
```

### Codex — if available

```
/codex:rescue --model gpt-5.4 --effort high --background \
  "{prompt} $(cat /tmp/tri-review-diff.txt)"
```

Retrieve result with `/codex:result` when done.

### Gemini — if available

```bash
if [ "$HAS_GEMINI" = "yes" ]; then
  gemini -p "{prompt} $(cat /tmp/tri-review-diff.txt)" \
    -m gemini-3.1-pro -y --output-format text \
    > /tmp/tri-review-gemini.txt 2>/dev/null &
  GEMINI_PID=$!
fi

wait
```

## Autonomy Flags

| Agent | Method | Flags |
|---|---|---|
| Claude | Native background agent | `isolation: worktree`, `background: true` |
| Codex | Official plugin | `/codex:rescue --background` |
| Gemini | CLI | `-y` |

## Step 5: Consolidate

```
## Triple-Agent Review: {branch_name}

### Consensus (flagged by 2+ agents — high confidence)
- ...

### Unique Findings (one agent only — worth investigating)
- Claude: ...
- Codex: ...
- Gemini: ...
```

## Presets

- `/tri:review` — full default review
- `/tri:review check for DRY violations` — custom prompt
- `/tri:review security focus` — security-oriented

## Rules

- Claude always runs as native background agent (not `claude -p`) for token efficiency
- Codex and Gemini are optional — run if installed, skip gracefully if not
- Report which agents participated and how many perspectives were gathered
- If only Claude is available, still provide the full report format
- If one agent fails, report the others
- For large diffs (>5000 lines), warn and suggest specific files
- Clean up temp files after
