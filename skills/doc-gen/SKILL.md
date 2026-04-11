---
name: doc-gen
description: Generate documentation for code — use when asked to document a module, generate API docs, create a README for code, or write reference documentation. Triggers the deterministic doc-gen workflow (analyze → generate via documenter agent → write).
---

# Documentation Generation

Deterministic doc generation. Analyze target → generate via the documenter subagent → write to docs/ or specified path.

## Invoke

Use the `devkit_start` tool with workflow: "doc-gen" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
