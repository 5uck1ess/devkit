---
description: Triple-agent debugging — independent root-cause hypotheses from Claude, Codex, and Gemini, then consensus fix.
---

## Invoke

Start the workflow via the devkit engine:

Use the `devkit_start` tool with workflow: "tri-debug" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
