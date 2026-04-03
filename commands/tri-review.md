---
name: tri:review
description: Triple-agent PR/code review. Claude runs as native background agent (token-efficient), Codex and Gemini via plugin or CLI. Consolidates findings.
---

# Triple-Agent Review

Run the same code review across three AI agents in parallel and consolidate results.

## Step 1: Gather Context

```bash
# Try branch diff first, fall back to last commit, then cached
DIFF=$(git diff main...HEAD 2>/dev/null)
if [ -z "$DIFF" ]; then DIFF=$(git diff HEAD~1 2>/dev/null); fi
if [ -z "$DIFF" ]; then DIFF=$(git diff --cached 2>/dev/null); fi

echo "$DIFF" > /tmp/tri-review-diff.txt
```

If all empty, ask the user what to review.

**CRITICAL:** The diff MUST be passed inline in each agent's prompt — do NOT rely on agents fetching the diff themselves. Worktree-isolated agents cannot see the latest commits.

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

Check for plugins first (preferred), then fall back to CLI:

```bash
# Plugin detection (preferred — structured job management)
HAS_CODEX_PLUGIN=$(/codex:status >/dev/null 2>&1 && echo "yes" || echo "no")
HAS_GEMINI_PLUGIN=$(/gemini:status >/dev/null 2>&1 && echo "yes" || echo "no")

# CLI fallback detection
HAS_CODEX_CLI=$(command -v codex && echo "yes" || echo "no")
HAS_GEMINI_CLI=$(command -v gemini && echo "yes" || echo "no")
```

Run with whatever is available. Claude always runs. Codex and Gemini are optional. Prefer plugin over CLI.

## Concurrency & Budget

- **Concurrency limit:** Max 3 parallel agents. All dispatches below run concurrently.
- **Token budget:** ~300k tokens across all agents.
- **Rate limiting:** If you hit API rate limits, wait and retry. Don't launch all agents simultaneously if the API is throttling.

## Step 4: Dispatch in Parallel (Hybrid, Graceful Degradation)

**[PARALLEL]** Launch all available agents concurrently:

### Claude — always runs (native background agent, token-efficient)

Spawn the `reviewer` agent as a background task. **Pass the full diff inline in the prompt** — the agent runs in a worktree and cannot see recent commits.

```
Task: Review this code diff.
Agent: reviewer
Input: {prompt}

```diff
{paste the actual diff content here — do NOT tell the agent to run git diff}
```
```

### Codex — if available

```
/codex:rescue --effort high --background \
  "{prompt} $(cat /tmp/tri-review-diff.txt)"
```

Retrieve result with `/codex:result` when done. Omit `--model` to use the account default.

### Gemini — if available

**Plugin (preferred):**

```
/gemini:rescue --background \
  "{prompt} $(cat /tmp/tri-review-diff.txt)"
```

Retrieve result with `/gemini:result` when done. Omit `--model` to use the account default.

**CLI fallback (only if plugin not installed):**

```bash
if [ "$HAS_GEMINI_CLI" = "yes" ]; then
  gemini -p "{prompt} $(cat /tmp/tri-review-diff.txt)" \
    -y --output-format text \
    > /tmp/tri-review-gemini.txt 2>/dev/null &
  GEMINI_PID=$!
fi

wait
```

Note: Gemini CLI defaults to the best available model. Don't hardcode a model name — it may not be available on all accounts.

## Autonomy Flags

| Agent | Method | Flags |
|---|---|---|
| Claude | Native background agent | `isolation: worktree`, `background: true` |
| Codex | Plugin (preferred) / CLI fallback | `/codex:rescue --background` or `codex -q` |
| Gemini | Plugin (preferred) / CLI fallback | `/gemini:rescue --background` or `-y` |

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
