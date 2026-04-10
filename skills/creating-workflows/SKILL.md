---
name: creating-workflows
description: How to create devkit workflow YAML files — schema reference, step types, variable interpolation, and examples.
---

# Writing Workflows

Devkit workflows are YAML files in the `workflows/` directory. Each workflow defines a sequence of steps executed in order, with optional loops, branches, and parallel groups.

## Schema

```yaml
name: string          # Workflow display name
description: string   # What this workflow does
budget:               # Optional: token budget
  limit: number       # Max tokens (e.g., 300000)
  downgrade: string   # Model to downgrade to when approaching limit
steps:
  - id: string        # Unique step identifier
    model: string     # Model tier: "smart", "general", "fast"
    prompt: string    # Instruction (supports {{variable}} interpolation)
    command: string   # Shell command — runs directly, no LLM (mutually exclusive with prompt)
    parallel: [ids]   # Optional: run these step IDs concurrently
    loop:             # Optional: repeat this step
      max: number     # Maximum iterations
      until: string   # Stop condition (string match in output)
      gate: string    # Shell command run after each iteration — exit 0 keeps, non-zero reverts
    branch:           # Optional: conditional execution
      if: string      # Condition expression
      then: string    # Step id to jump to if true
      else: string    # Step id to jump to if false
```

## Step Fields

- **id** — Must be unique. Used for branch targets and output references.
- **model** — Which model tier runs this step. Pick based on task complexity.
- **prompt** — The instruction. Use `{{step-id}}` to reference earlier outputs. Use `{{input}}` for workflow input. Mutually exclusive with `command`.
- **command** — Shell command run directly (no LLM). Output is captured and available via `{{step-id}}`. Costs $0. Mutually exclusive with `prompt`.
- **expect** — Only for `command` steps. Values: `success` (step fails if exit code is non-zero) or `failure` (step fails if exit code is 0). Omit for the default behavior where all exit codes are informational. Enables bugfix reproduction gates: repro with `expect: failure` must fail before fix, verify with `expect: success` must pass after.
- **parallel** — Lists step IDs to run concurrently. Results collected before next sequential step.
- **loop** — Repeats with `max` iterations. Exits early if output contains `until` string. Optional `gate` command enforces quality after each iteration.
- **loop.gate** — Shell command run after each loop iteration. Exit 0 = keep changes and commit. Non-zero = revert changes via `git checkout`. 3 consecutive gate failures trigger stuck detection and stop the loop.
- **branch** — Routes execution conditionally. Both `then` and `else` reference step `id`s.

## Minimal Example

```yaml
name: summarize-and-review
description: Summarize a document then review the summary
steps:
  - id: summarize
    model: general
    prompt: |
      Summarize in 3 bullet points:
      ${{input.document}}

  - id: review
    model: smart
    prompt: |
      Review this summary for accuracy:
      Summary: ${{steps.summarize.output}}
      Original: ${{input.document}}
```

## Command + Gate Example

```yaml
name: lint-and-fix
description: Deterministic lint loop with gate enforcement
steps:
  - id: baseline
    command: "eslint src/ 2>&1 || true"

  - id: fix
    model: smart
    prompt: |
      Lint output: {{baseline}}
      Fix ONE group of related issues.
    loop:
      max: 10
      until: "exit code: 0"
      gate: "eslint src/"

  - id: report
    command: "echo 'Lint session complete'"
```

Key behaviors:
- `command` steps run shell commands directly — no LLM tokens spent.
- `gate` runs after each loop iteration. Exit 0 keeps changes, non-zero reverts via git.
- 3 consecutive gate failures stop the loop (stuck detection).
- Command output includes `exit code: N` for downstream branching.

## Tips

- Keep prompts focused. One task per step.
- Use descriptive `id`s — they appear in logs and output references.
- Put complex multi-step logic in workflows, not in prompts.
- Test workflows with small inputs first.
- Set token budgets for expensive workflows to prevent runaway costs.
