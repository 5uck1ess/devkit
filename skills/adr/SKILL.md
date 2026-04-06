---
name: adr
description: Generate Architecture Decision Records — use when asked to document a decision, create an ADR, record why we chose X, or capture architectural rationale.
---

# ADR — Architecture Decision Record

Capture the *why* behind architectural decisions so future-you (and future-agents) don't reverse them without context.

## Step 1: Gather Context

Identify:
- What decision was made (or needs to be made)
- What alternatives were considered
- What constraints drove the choice
- What the consequences are

If the user hasn't specified, ask one question: "What decision are you documenting?"

## Step 2: Check Existing ADRs

```bash
mkdir -p docs/adr
ls docs/adr/ 2>/dev/null | grep -oE '^[0-9]+' | sort -n | tail -1
```

Add 1 to the highest number found, zero-padded to 4 digits. If none exist, start at `0001`.

## Step 3: Write the ADR

Create `docs/adr/NNNN-short-title.md`:

```markdown
# NNNN. Short Decision Title

**Date:** YYYY-MM-DD
**Status:** accepted | proposed | deprecated | superseded by [NNNN]

## Context

What is the issue? What forces are at play? 2-4 sentences max.

## Decision

What did we decide? State it directly. 1-3 sentences.

## Alternatives Considered

- **Alternative A** — why rejected (1 line)
- **Alternative B** — why rejected (1 line)

## Consequences

What follows from this decision? Both positive and negative. Bullet list.
```

## Rules

- Keep it short. An ADR is a reference, not an essay. Under 200 words total.
- One decision per ADR. If there are two decisions, write two ADRs.
- Use plain language. No jargon that requires context to parse.
- Status is usually `accepted`. Use `proposed` only if the user hasn't decided yet.
- Never modify existing ADRs — create a new one that supersedes. When superseding, update the old ADR's status line to `superseded by [MMMM]`.
