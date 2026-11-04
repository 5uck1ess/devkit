---
name: creating-workflows
description: REFERENCE ONLY — schema documentation for authoring devkit workflow YAML files (step types, variable interpolation, loops, parallel groups, examples). NOT a runnable workflow. Load this only when asked to write or modify a workflow YAML, never on natural-language task requests.
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
    branch:           # Optional: first matching rule wins
      - when: string  # Sentinel word to look for in this step's output
        goto: string  # Step id to jump to
    end: bool         # Optional: workflow completes after this step (unless a branch matched)
```

## Step Fields

- **id** — Must be unique. Used for branch targets and output references.
- **model** — Which model tier runs this step. Pick based on task complexity.
- **prompt** — The instruction. Use `{{step-id}}` to reference earlier outputs. Use `{{input}}` for workflow input. Mutually exclusive with `command`.
- **command** — Shell command run directly (no LLM). Output is captured and available via `{{step-id}}`. Costs $0. Mutually exclusive with `prompt`.
- **expect** — Only for `command` steps. Values: `success` (step fails if exit code is non-zero) or `failure` (step fails if exit code is 0). Omit for the default behavior where all exit codes are informational. Enables bugfix reproduction gates: repro with `expect: failure` must fail before fix, verify with `expect: success` must pass after.
- **parallel** — Lists step IDs to run concurrently. Listed steps are skipped in their sequential position and run only inside the dispatcher. Children must be plain prompt steps (no command, loop, branch, or end), and a branch may not `goto` a child. Over MCP, the agent reports results with `devkit_advance` `outputs: {"<child-id>": "<result>", ...}` — one entry per child — so later steps can use `{{child-id}}`.
- **loop** — Repeats with `max` iterations. Completes when the `gate` (if set) passes **and** the `until` sentinel (if set) appears in the iteration's output — a failing gate is never overridden by the sentinel. Over MCP, if `max` is reached with the gate still failing the workflow halts (`loop_exhausted`); if only the sentinel is missing it continues with a `LOOP EXHAUSTED` warning.
- **loop.gate** — Shell command run after each loop iteration. Exit 0 = keep changes and commit. Non-zero = revert changes via `git checkout`. 3 consecutive gate failures trigger stuck detection and stop the loop.
- **branch** — Routes execution conditionally: a list of `when`/`goto` rules evaluated against the step's output; the first match jumps to `goto`, no match falls through.
- **end** — Marks a terminal step. After it runs (and no branch rule matched) the workflow completes instead of falling through. Use it to stop a main path before an alternative path placed later in the file (e.g. a TINY `quick-fix` reached only by branch).

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
