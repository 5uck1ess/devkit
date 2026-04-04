---
description: Run a user-defined YAML workflow. Multi-step pipelines with loops, approval gates, and branching.
---

# Workflow Runner

Execute a YAML-defined workflow from the `workflows/` directory.

## Usage

```
/devkit:workflow {name}
/devkit:workflow list
```

## Step 1: Load Workflow

Read `workflows/{name}.yml` and parse:

```yaml
description: What this workflow does

steps:
  - id: step-1
    prompt: "Do the first thing"
    agent: improver          # optional — which agent to use
    approval: false          # optional — pause for user approval before executing

  - id: step-2
    prompt: "Do the second thing based on: {{step-1}}"
    loop:
      max: 5                 # max iterations
      until: "all passing"   # stop when output contains this

  - id: step-3
    prompt: "Finalize"
    branch:                  # conditional jump
      - when: "error"
        goto: step-1
      - when: "success"
        goto: done
```

## Step 2: Execute Steps

For each step, sequentially:

1. **Resolve placeholders** — replace `{{step-id}}` with that step's output, `{{input}}` with user's original input
2. **Approval gate** — if `approval: true`, show the step prompt and ask user to confirm before running
3. **Execute** — run the prompt (optionally via the specified agent)
4. **Loop** — if `loop` defined, repeat until output contains `until` string or `max` reached
5. **Branch** — if `branch` defined, check output for `when` string and jump to `goto` step
6. **Store result** — save output for placeholder resolution in later steps

## Step 3: Report

```
## Workflow: {name}

### Steps Completed
| Step | Status | Iterations |
|------|--------|------------|
| step-1 | ✓ completed | 1 |
| step-2 | ✓ completed | 3 (loop) |
| step-3 | ✓ completed | 1 |

### Output
{final_step_output}
```

## Listing Workflows

When called with `list`, scan `workflows/` and display:

```
## Available Workflows

| Name | Description |
|------|-------------|
| my-workflow | Does something useful |
```

## Rules

- Steps execute sequentially — no parallel steps
- Placeholder `{{step-id}}` resolves to that step's output
- Loop aborts if step executed more than `loop.max` times
- Branch checks are case-insensitive substring matches
- If no agent specified, execute the prompt directly (main Claude context)
- If a step fails, stop the workflow and report which step failed
- Approval gates require explicit user confirmation
