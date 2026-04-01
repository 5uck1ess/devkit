# Devkit Agent Preferences

Loaded as context for all devkit agents. Defines communication style and coding standards.

## Communication

- Lead with the answer. No preamble, no trailing summary.
- Don't add what wasn't asked for — no bonus docstrings, annotations, or refactors.
- Three similar lines beats a premature abstraction.
- Confirm before destructive operations.

## Standards

- **TypeScript:** strict, no `any`, const-first
- **Python:** typed signatures, f-strings
- **Go:** stdlib-first
- **Shell:** `set -euo pipefail`, quoted vars, `[[ ]]`
- Functions fit on one screen. Errors say what broke and how to fix it.

## Git

- Imperative commit messages, under 72 chars, no trailing period
- One logical change per commit
- No amending published commits without asking

## Avoid

- Emojis, time estimates, backwards-compat shims
- Creating files unless necessary
- Re-exporting or renaming unused code — delete it

## Model Tier Guidance

When commands reference model tiers:

| Tier | When to Use | Devkit Agent Examples |
|------|------------|---------------------|
| **Smart** (Opus) | Complex analysis, security audit, architectural review | reviewer, improver, security-auditor |
| **General** (Sonnet) | Balanced tasks, test writing, research, implementation | researcher, test-writer |
| **Fast** (Haiku) | Summaries, simple checks, documentation, formatting | documenter |

Use the cheapest tier that can do the job. Upgrade only when quality matters more than cost.
