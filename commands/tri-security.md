---
description: Multi-agent security audit — independent security reviews from available agents, consolidated with severity-ranked findings.
---

# Triple-Agent Security Audit

Independent security reviews from all available agents, consolidated into a severity-ranked report.

## Step 0: Harness Detection

```bash
if command -v devkit >/dev/null 2>&1; then
  echo "Go harness detected — delegating to devkit review --security for full output capture."
  devkit review --security {prompt or default}
  exit 0
fi
```

If the `devkit` binary is in PATH, delegate entirely to it. Only fall through to plugin-based steps if the harness is not installed.

## Step 1: Gather Scope

Determine what to audit:
- If args specify files/dirs, use those
- If on a branch with changes, audit the diff: `git diff main...HEAD`
- Otherwise, audit the full project

```bash
# Write directly to file — avoids shell variable limits
git diff main...HEAD > /tmp/tri-security-diff.txt 2>/dev/null
if [ ! -s /tmp/tri-security-diff.txt ]; then git diff HEAD~1..HEAD > /tmp/tri-security-diff.txt 2>/dev/null; fi
if [ ! -s /tmp/tri-security-diff.txt ]; then git diff --cached > /tmp/tri-security-diff.txt 2>/dev/null; fi
```

**CRITICAL:** All code/diff MUST be passed inline in each agent's prompt. Worktree-isolated agents cannot see the latest commits.

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

Prefer plugin over CLI.

## Step 3: Build the Prompt

```
Perform a security audit of this code. Check for:

1. **Injection** — SQL injection, command injection, XSS, template injection
2. **Authentication/Authorization** — broken auth, missing access controls, privilege escalation
3. **Secrets** — hardcoded credentials, API keys, tokens in source
4. **Data Exposure** — sensitive data in logs, responses, or error messages
5. **Dependencies** — known vulnerable packages
6. **Cryptography** — weak algorithms, improper key handling
7. **Configuration** — debug mode in prod, permissive CORS, missing security headers
8. **Input Validation** — missing or insufficient validation at boundaries

For each finding, report:
- Severity: CRITICAL / HIGH / MEDIUM / LOW
- Location: file and line number
- Description: what the vulnerability is
- Impact: what an attacker could do
- Fix: specific code change to remediate

Code: {diff_or_source}
```

## Concurrency & Budget

- **Concurrency limit:** Max 3 parallel agents.
- **Token budget:** ~300k tokens across all agents.
- **Rate limiting:** If API throttles, stagger agent launches.

## Step 4: Dispatch (Hybrid, Graceful Degradation)

**[PARALLEL]** Launch all available agents concurrently:

### Claude — always runs

Pass the code/diff inline — the agent runs in a worktree and cannot see recent commits.

```
Task: Security audit of this code.
Agent: security-auditor
Input: {prompt}

```diff
{diff}
```
```

<!-- The orchestrator MUST inline the diff/code here. The agent runs in a worktree and cannot fetch it. -->

### Codex — if available

```
/codex:rescue --effort high --background "{prompt} $(cat /tmp/tri-security-diff.txt)"
```

Retrieve result with `/codex:result` when done. Omit `--model` to use the account default.

### Gemini — if available

**Plugin (preferred):**

```
/gemini:rescue --background "{prompt} $(cat /tmp/tri-security-diff.txt)"
```

Retrieve result with `/gemini:result` when done. Omit `--model` to use the account default.

**CLI fallback (only if plugin not installed):**

```bash
if [ "$HAS_GEMINI_CLI" = "yes" ]; then
  cat /tmp/tri-security-diff.txt | gemini -p "{prompt}" -y \
    --output-format text > /tmp/tri-security-gemini.txt 2>&1 &
fi

wait
```

## Step 5: Consolidate

```
## Security Audit Report

### Agents Used: {count}/3
### Scope: {files_or_diff_range}

### Critical / High Findings (consensus — flagged by 2+ agents)
| # | Severity | Location | Finding | Agents |
|---|----------|----------|---------|--------|
| 1 | CRITICAL | src/api.ts:42 | SQL injection in query builder | Claude, Codex, Gemini |
| 2 | HIGH | src/auth.ts:15 | Missing rate limiting on login | Claude, Gemini |

### Medium / Low Findings
| # | Severity | Location | Finding | Agent |
|---|----------|----------|---------|-------|
| 3 | MEDIUM | config.ts:8 | Debug mode flag checked via env var | Claude |
| 4 | LOW | utils.ts:22 | Overly permissive regex | Codex |

### Unique Findings (single agent — worth investigating)
- **Claude:** ...
- **Codex:** ...
- **Gemini:** ...

### Summary
- Critical: {n}
- High: {n}
- Medium: {n}
- Low: {n}
- **Consensus findings (high confidence):** {n}
```

## Rules

- Claude always runs — others are optional
- Consensus findings (2+ agents) ranked higher than single-agent findings
- Sort by severity, then by consensus count
- Include specific file:line references
- Provide actionable fix for each finding
- Clean up temp files after
