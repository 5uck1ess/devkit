---
description: Triple-agent test generation — each agent generates tests independently, then merge for maximum coverage.
---

# Triple-Agent Test Generation

Dispatch test generation to 2-3 AI agents in parallel, then merge and deduplicate for maximum coverage.

## Steps

1. **Analyze target** — Read source code, identify public API, detect test framework and conventions
2. **Scenario expansion** — Generate test scenarios using: boundary values, equivalence partitioning, error injection, state transitions, concurrency (if applicable)
3. **Detect agents** — Check for Codex and Gemini availability. Claude always runs.
4. **Dispatch in parallel** — Each agent generates tests independently for the target
5. **Merge & deduplicate** — Combine test files, remove duplicates, resolve naming conflicts
6. **Run & fix** — Execute merged tests, fix any failures
7. **Report** — Coverage summary, test count per agent, merged result

## Rules

- Claude always runs — others optional
- Each agent generates independently — no shared context between agents
- Merge by deduplicating equivalent test cases
- All merged tests must pass before reporting
