---
description: Full lifecycle feature development — brainstorm, plan, implement, test, lint, review.
---

# Feature Workflow

Complete feature lifecycle: brainstorm → plan → implement → test → lint → review → report.

## Step 1: Brainstorm

```
Feature: {user's input}

Think through the design:
- What components need to change?
- What's the simplest approach that works?
- What are the risks or unknowns?
- Are there edge cases to handle upfront?

Produce a short design summary. Don't write code yet.
```

## Step 2: Plan

```
Based on the design from Step 1, create an implementation plan
as a numbered todo list.

Each item should be a single, testable change.
Order by dependency — do foundations first.
Include a final item for writing tests.
```

## Step 3: Implement

```
Execute the next incomplete todo from the plan.
Write the code, verify it works, then mark it done.
Keep changes small and focused.
```

Loop until all todos are complete. Max 20 iterations.

## Step 4: Generate Tests

```
Generate tests covering:
- Happy path for each new public function/endpoint
- Edge cases and error conditions
- Integration between new and existing code

Use the project's existing test framework.
Place tests in the project's standard test location.
```

## Step 5: Run Tests

Run the full test suite (not just the new tests). If tests fail, fix them — determine if the bug is in the test or the implementation. Loop up to 8 times until all pass.

## Step 6: Lint

Run the project's linter on changed files. If there are violations, fix them without changing code behavior. Loop up to 4 times until clean.

## Budget

- **Token budget:** ~500k tokens. Features are the most expensive workflow.
- If approaching budget, skip the review step and report what was completed.

## Step 7: Review

**[PARALLEL]** Spawn the `reviewer` agent to review all changes (can run concurrently with any remaining lint fixes):

```
Task: Review all changes made in this feature session.
Agent: reviewer
Context:
  - Design intent from brainstorm step
  - Check for: correctness, security issues, missing error handling,
    performance problems, violations of original design intent
  - Be specific — reference files and line numbers
```

## Step 8: Final Report

```
## What was built
Design summary from brainstorm.

## Implementation
Brief description of what was implemented and how many steps it took.

## Test coverage
Test results summary.

## Review findings
Issues found during review.

## Status
Ready to commit, or list remaining issues.
```

## Rules

- Design before code — brainstorm and plan first, implement second
- One todo at a time — finish each step before starting the next
- Tests are not optional — every feature gets tests
- Lint is not optional — clean code before review
- Review before done — catch issues before they're committed
- Loop on failures — don't give up after one test failure
