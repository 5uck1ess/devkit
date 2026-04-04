---
name: devkit:creating-workflows
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
    prompt: string    # Instruction (supports ${{variable}} interpolation)
    parallel: [ids]   # Optional: run these step IDs concurrently
    loop:             # Optional: repeat this step
      max: number     # Maximum iterations
      until: string   # Stop condition (string match in output)
    branch:           # Optional: conditional execution
      if: string      # Condition expression
      then: string    # Step id to jump to if true
      else: string    # Step id to jump to if false
```

## Step Fields

- **id** — Must be unique. Used for branch targets and output references.
- **model** — Which model tier runs this step. Pick based on task complexity.
- **prompt** — The instruction. Use `${{steps.previous_id.output}}` to reference earlier outputs. Use `${{input.field}}` for workflow inputs.
- **parallel** — Lists step IDs to run concurrently. Results collected before next sequential step.
- **loop** — Repeats with `max` iterations. Exits early if output contains `until` string.
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

## Tips

- Keep prompts focused. One task per step.
- Use descriptive `id`s — they appear in logs and output references.
- Put complex multi-step logic in workflows, not in prompts.
- Test workflows with small inputs first.
- Set token budgets for expensive workflows to prevent runaway costs.
