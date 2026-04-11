---
name: test-gen
description: Generate tests for code — use when asked to write tests, create a test suite, add test coverage, or generate unit/integration tests for a file or module. Triggers the deterministic test-gen workflow (analyze → generate via test-writer agent → run-fix loop → report).
---

# Test Generation

Deterministic test generation. Analyze target → generate via the test-writer subagent → run tests and iterate up to 3x → report.

## Invoke

Use the `devkit_start` tool with workflow: "test-gen" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
