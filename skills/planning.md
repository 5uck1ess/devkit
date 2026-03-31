---
name: devkit:planning
description: Write implementation plans — small testable steps, dependency ordering, upfront risk identification.
---

# Planning

## What a Good Plan Looks Like

A plan is an ordered list of small steps. Each step should be:

- **Independently testable** — you can verify it works before moving on
- **Small** — completable in one focused session
- **Concrete** — "Add validation to `parseConfig`" not "improve error handling"

## Breaking Down Work

Start from the end: What does "done" look like? Work backwards.

1. Identify the deliverable
2. List the components/changes needed
3. For each component, identify the smallest meaningful increment
4. Order by dependency (what must exist before what?)

If a step feels large or vague, break it down further. If you can't describe how to verify a step, it's too abstract.

## Ordering Steps

- **Dependencies first**: Build the foundation before the features
- **Riskiest first**: Tackle unknowns early — if something will blow up the plan, find out now
- **Vertical slices over horizontal layers**: Prefer "auth end-to-end" over "all database tables, then all API routes, then all UI"

## Identifying Risks

Before starting, ask:

- What am I least sure about technically?
- Where am I making assumptions I haven't verified?
- What external dependencies could block me?
- What's the hardest part? (Do that first.)

Flag risks explicitly in the plan. A risk without a mitigation is just a wish.

## Plan Format

```
## Plan: [Feature Name]

### Steps
1. [ ] Step description — how to verify
2. [ ] Step description — how to verify
3. [ ] Step description — how to verify

### Risks
- Risk description → mitigation
```

## Plans Change

A plan is a starting hypothesis, not a contract. Update it as you learn. The goal is to think through the work upfront, not to predict the future perfectly.
