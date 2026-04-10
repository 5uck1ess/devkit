---
description: Triple-agent code review — dispatches to Claude, Codex, and Gemini in parallel, consolidates findings.
---

## Invoke

Start the workflow via the devkit engine:

Use the `devkit_start` tool with workflow: "tri-review" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
