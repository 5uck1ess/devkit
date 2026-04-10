---
description: Triple-agent security audit — independent security reviews from Claude, Codex, and Gemini, consolidated with severity ranking.
---

## Invoke

Start the workflow via the devkit engine:

Use the `devkit_start` tool with workflow: "tri-security" and input: "{input}".

Then follow each step the engine returns. Call `devkit_advance` after completing each step. The engine controls step order, gates, and loops. Do NOT skip steps.
