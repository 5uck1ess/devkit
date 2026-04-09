# Deterministic Workflow Conversion

Convert all rigid commands from LLM-interpreted markdown to Go-engine-driven YAML workflows. Same steps, same results, same triggers — Claude can't skip steps.

## Problem

24 markdown command files define multi-step procedures. Claude interprets these as prompts and routinely:
- Skips verification steps ("I already know this")
- Fabricates baselines instead of running tools
- Jumps to workarounds when a step fails instead of retrying correctly
- Skips disconfirmation in research (confirms own hypothesis)
- Presents results without running the actual commands

## Solution

Replace markdown command logic with YAML workflows executed by the Go engine. The engine owns the sequence — `command` steps run shell commands deterministically, `gate` checks enforce quality after each loop iteration, and Claude only handles the thinking within each step.

## Engine Addition

One new primitive needed:

### `expect` field on command steps

```yaml
- id: repro
  command: "{{input}}"
  expect: failure  # step fails if exit code is 0
```

Values: `success` (default — non-zero exit is informational), `failure` (step fails if exit code is 0). Enables bugfix reproduction gates: repro must fail before fix, pass after.

## Conversion Plan

### PR 1: Research workflows

**research.yml**
```yaml
steps:
  - id: clarify
    model: smart
    prompt: |
      Clarify the research question. Identify 3-5 sub-questions.
      {{input}}

  - id: search
    model: smart
    prompt: |
      Search for answers to each sub-question. Use web search, grep,
      and file reads. Do NOT answer from memory.
      
      Sub-questions: {{clarify}}
      
      For each finding, cite the source.
    loop:
      max: 5
      until: SUFFICIENT_EVIDENCE

  - id: corroborate
    model: smart
    prompt: |
      Cross-check findings against 2+ independent sources.
      Flag anything with only one source.
      
      Findings: {{search}}

  - id: synthesize
    model: smart
    prompt: |
      Synthesize findings into a clear answer.
      Lead with the conclusion, then supporting evidence.
      
      Corroborated findings: {{corroborate}}
```

**deep-research.yml**
```yaml
steps:
  - id: clarify
    model: smart
    prompt: |
      Clarify the research question. Identify perspectives that
      might disagree. {{input}}

  - id: search
    model: smart
    prompt: |
      Search exhaustively. Use web search, grep, file reads.
      Do NOT answer from memory.
      
      Question: {{clarify}}
    loop:
      max: 8
      until: SUFFICIENT_EVIDENCE

  - id: hypotheses
    model: smart
    prompt: |
      Form 2-3 competing hypotheses from the evidence.
      {{search}}

  - id: disconfirm
    model: smart
    prompt: |
      For EACH hypothesis, actively search for evidence that
      DISPROVES it. Do not confirm — try to break each one.
      
      Hypotheses: {{hypotheses}}
    loop:
      max: 5
      until: DISCONFIRMATION_COMPLETE

  - id: matrix
    model: smart
    prompt: |
      Build an evidence matrix: hypotheses as columns, evidence
      as rows. Mark consistent/inconsistent/neutral.
      
      Evidence: {{search}}
      Disconfirmation: {{disconfirm}}

  - id: synthesize
    model: smart
    prompt: |
      Synthesize. Which hypothesis survives disconfirmation best?
      Rate confidence. Flag remaining uncertainties.
      
      Matrix: {{matrix}}
```

### PR 2: Self-improvement loops

All follow the same pattern — `command` step for baseline, `gate` on the loop:

**self-test.yml** (example — others are identical pattern)
```yaml
steps:
  - id: baseline
    command: "{{input}} 2>&1 || true"

  - id: improve
    model: smart
    prompt: |
      Current test output:
      {{baseline}}

      Generate or improve tests to increase coverage.
      Focus on untested code paths and edge cases.
      ONE test file at a time.
    loop:
      max: 10
      until: "exit code: 0"
      gate: "{{input}}"

  - id: verify
    command: "{{input}} 2>&1 || true"

  - id: summary
    model: fast
    prompt: |
      Test improvement session complete.
      Before: {{baseline}}
      After: {{verify}}
      Summarize what was added.
```

**self-perf.yml**, **self-migrate.yml**, **self-improve.yml** — same structure, different prompts within each step.

### PR 3: Lifecycle gates

**bugfix.yml**
```yaml
steps:
  - id: repro
    command: "{{input}} 2>&1 || true"

  - id: diagnose
    model: smart
    prompt: |
      Bug reproduction output:
      {{repro}}

      Diagnose the root cause. Read relevant source files.
      Identify the exact location of the bug.

  - id: fix
    model: smart
    prompt: |
      Root cause: {{diagnose}}

      Fix the bug. Minimal change only.
      Don't refactor surrounding code.

  - id: verify
    command: "{{input}} 2>&1 || true"

  - id: check
    model: fast
    prompt: |
      Before fix: {{repro}}
      After fix: {{verify}}

      Did the fix resolve the bug? Say FIXED or NOT_FIXED.
    branch:
      - when: NOT_FIXED
        goto: diagnose
      - when: FIXED
        goto: summary

  - id: summary
    model: fast
    prompt: |
      Bug fix complete.
      Reproduction: {{repro}}
      Diagnosis: {{diagnose}}
      Verification: {{verify}}
      Summarize what was wrong and what was changed.
```

**feature.yml**
```yaml
steps:
  - id: explore
    model: smart
    prompt: |
      Explore the codebase to understand relevant patterns,
      conventions, and architecture. Identify 5-10 key files.
      {{input}}

  - id: design
    model: smart
    prompt: |
      Based on codebase exploration:
      {{explore}}

      Propose 2-3 design approaches with trade-offs.
      Recommend one. Include data flow and component boundaries.

  - id: plan
    model: smart
    prompt: |
      Design: {{design}}

      Create a numbered implementation plan.
      Order by dependency. Each step should be one logical change.

  - id: implement
    model: smart
    prompt: |
      Plan: {{plan}}

      Implement the next unfinished step.
      Small, focused changes. Follow existing patterns.
    loop:
      max: 15
      until: ALL_STEPS_COMPLETE

  - id: test
    model: smart
    prompt: |
      Implementation complete.

      Write tests for the new feature.
      Run them and fix any failures.
    loop:
      max: 5
      until: ALL_PASSING

  - id: summary
    model: fast
    prompt: |
      Feature complete.
      Design: {{design}}
      Implementation: {{implement}}
      Tests: {{test}}
      Summarize what was built.
```

### PR 4: Shipping + utility

**pr-ready.yml**
```yaml
steps:
  - id: lint
    command: "{{input}} 2>&1 || true"

  - id: lint-check
    model: fast
    prompt: |
      Lint output: {{lint}}
      Are there errors? Say CLEAN or HAS_ERRORS.
    branch:
      - when: HAS_ERRORS
        goto: lint-fix
      - when: CLEAN
        goto: test

  - id: lint-fix
    model: smart
    prompt: |
      Fix lint errors: {{lint}}
    loop:
      max: 5
      until: "exit code: 0"
      gate: "{{input}}"

  - id: test
    command: "{{test_command}} 2>&1 || true"

  - id: security
    model: smart
    prompt: |
      Review changed files for security issues.
      Check OWASP top 10 patterns.

  - id: changelog
    model: fast
    prompt: |
      Generate changelog entry from git diff.

  - id: create-pr
    model: smart
    prompt: |
      Create the PR with changelog and summary.
```

**audit.yml** — all `command` steps for tool execution:
```yaml
steps:
  - id: detect
    command: |
      echo "go:$(test -f go.mod && echo yes || echo no)"
      echo "node:$(test -f package.json && echo yes || echo no)"
      echo "python:$(test -f requirements.txt -o -f pyproject.toml && echo yes || echo no)"
      echo "rust:$(test -f Cargo.toml && echo yes || echo no)"

  - id: deps
    model: smart
    prompt: |
      Detected ecosystems: {{detect}}
      Run dependency audit commands for each detected ecosystem.
      Report vulnerabilities, outdated packages, and license issues.

  - id: lint
    model: smart
    prompt: |
      Run linters for detected ecosystems: {{detect}}

  - id: report
    model: fast
    prompt: |
      Compile audit report.
      Dependencies: {{deps}}
      Lint: {{lint}}
      Score overall health.
```

**tri-review.yml**, **tri-debug.yml**, **tri-security.yml**, **tri-dispatch.yml**, **tri-test-gen.yml** — add `command` step to capture diff/context deterministically before dispatch.

### PR 5: Trim commands + thin wrappers + docs

**Delete** these markdown command files (logic lives in YAML):
- autoloop.md, bugfix.md, deep-research.md, feature.md, refactor.md
- self-audit.md, self-improve.md, self-lint.md, self-perf.md, self-test.md, self-migrate.md
- tri-debug.md, tri-dispatch.md, tri-review.md, tri-security.md, tri-test-gen.md
- audit.md, decompose.md, pr-ready.md, repo-map.md

**Keep as thin wrappers** (tab-completable, one-liner pointing to workflow):
- `tri-review.md` → "Run `devkit workflow tri-review`"
- `tri-debug.md` → "Run `devkit workflow tri-debug`"
- `tri-security.md` → "Run `devkit workflow tri-security`"
- `pr-ready.md` → "Run `devkit workflow pr-ready`"
- `pr-monitor.md` → stays as-is (no YAML equivalent yet)

**Keep as-is** (not workflows):
- `status.md` — diagnostic
- `setup-rules.md` — one-time setup
- `workflow.md` — entry point

**Context-activated** (move trigger logic to `skills/`):
- research, deep-research, bugfix, feature, refactor, self-test, self-lint, self-improve, audit, decompose

**Docs updates:**
- README: Update to reflect ~8 slash commands
- creating-workflows skill: Document `expect` field
- ROADMAP: Add deterministic conversion as completed milestone

## What Does NOT Change

- 10 hooks (already deterministic shell scripts)
- 6 agents (used by workflows, not changed)
- Coding principle skills (clean-code, dry, yagni — judgment-based by design)
- Tool skills (gcli, creating-workflows)
- Companion plugins (superpowers, pr-review-toolkit, hookify, etc.)

## Success Criteria

- All 24 commands covered: converted to YAML, kept as thin wrapper, or kept as-is
- Zero duplicated logic between markdown and YAML
- All tests pass (`go test ./...`)
- All existing YAML workflows still parse (`TestParseRealWorkflows`)
- Tab-completion works for the ~8 kept commands
- Context-activation works for migrated workflows

## Token Efficiency

- `command` steps cost $0 (shell execution, no LLM)
- Baselines, linter runs, test runs, diff captures all move to `command` steps
- LLM only invoked for thinking steps (diagnosis, design, synthesis)
- Gate failures revert and retry — no tokens wasted on broken iterations
