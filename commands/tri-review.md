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

## Step 3: Dispatch in Parallel (Hybrid)

### Claude — native background agent (token-efficient)

Spawn the `reviewer` agent as a background task. The orchestrator only receives the summary, not the full conversation.

```
Task: Review this diff using the reviewer agent.
Agent: reviewer
Input: {prompt} + {diff}
```

### Codex — CLI dispatch

```bash
codex exec -m gpt-5.4 \
  --sandbox read-only \
  --full-auto \
  --skip-git-repo-check \
  --dangerously-bypass-approvals-and-sandbox \
  "{prompt} $(cat /tmp/tri-review-diff.txt)" > /tmp/tri-review-codex.txt 2>/dev/null &
CODEX_PID=$!
```

### Gemini — CLI dispatch

```bash
gemini -p "{prompt} $(cat /tmp/tri-review-diff.txt)" \
  -m gemini-3.1-pro -y --output-format text \
  > /tmp/tri-review-gemini.txt 2>/dev/null &
GEMINI_PID=$!

wait $CODEX_PID $GEMINI_PID
```

## Autonomy Flags

| Agent | Method | Flags |
|---|---|---|
| Claude | Native background agent | `isolation: worktree`, `background: true` |
| Codex | CLI | `--full-auto --dangerously-bypass-approvals-and-sandbox` |
| Gemini | CLI | `-y` |

## Step 4: Consolidate

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

- Claude uses native background agent (not `claude -p`) for token efficiency
- Codex and Gemini run as CLI background processes
- If one agent fails, report the others
- For large diffs (>5000 lines), warn and suggest specific files
- Clean up temp files after
