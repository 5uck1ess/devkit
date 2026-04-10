---
description: Full PR preparation pipeline — validate branch, DRY review, lint, test, security, changelog, create PR.
---

## Invoke

Start the workflow via the devkit engine:

Use the `devkit_start` tool with workflow: "pr-ready" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
