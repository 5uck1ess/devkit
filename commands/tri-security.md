---
name: tri:security
description: Multi-agent security audit — independent security reviews from available agents, consolidated with severity-ranked findings.
---

# Triple-Agent Security Audit

Independent security reviews from all available agents, consolidated into a severity-ranked report.

## Step 1: Gather Scope

Determine what to audit:
- If args specify files/dirs, use those
- If on a branch with changes, audit the diff: `git diff main...HEAD`
- Otherwise, audit the full project

```bash
git diff main...HEAD > /tmp/tri-security-diff.txt 2>/dev/null
```

## Step 2: Detect Available Agents

```bash
HAS_CODEX=$(command -v codex && echo "yes" || echo "no")
HAS_GEMINI=$(command -v gemini && echo "yes" || echo "no")
```

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

## Step 4: Dispatch (Hybrid, Graceful Degradation)

### Claude — always runs

```
Task: Security audit using the security-auditor agent.
Agent: security-auditor
Input: {prompt} + {code}
```

### Codex — if available

```
/codex:rescue --model gpt-5.4 --effort high --background "{prompt}"
```

Retrieve result with `/codex:result` when done.

### Gemini — if available

```bash
if [ "$HAS_GEMINI" = "yes" ]; then
  gemini -p "{prompt}" -m gemini-3.1-pro -y \
    --output-format text > /tmp/tri-security-gemini.txt 2>/dev/null &
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
