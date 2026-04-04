---
name: devkit:pr-monitor
description: Post-PR review monitor — watches CI, fetches reviewer comments, iteratively resolves them, and pushes fixes.
---

# PR Monitor

After a PR is created, this command watches for CI results and reviewer comments, then iteratively resolves them. Picks up where `/devkit:pr-ready` leaves off.

## Parameters

1. **PR number or URL** — the PR to monitor (required, or auto-detect from current branch)
2. **Max iterations** — max comment-resolution cycles (default: 10)
3. **Budget** — max USD (default: $5)

## Budget & Early Exit

- **Token budget:** ~500k tokens. Comment resolution can be expensive with large diffs.
- **Early exit:** Stop when all checks pass and no unresolved comments remain.
- **Stuck detection:** If 3 consecutive iterations resolve zero comments, stop and report.

## Step 1: Identify PR

```bash
# Auto-detect from current branch if no PR specified
PR_NUM=${1:-$(gh pr view --json number -q '.number' 2>/dev/null)}
if [ -z "$PR_NUM" ]; then
  echo "ERROR: No PR found for current branch. Specify a PR number."
  exit 1
fi

echo "Monitoring PR #${PR_NUM}"
gh pr view "$PR_NUM" --json title,state,statusCheckRollup,reviewDecision
```

## Step 2: Wait for Initial CI + Auto-Reviewers

Wait up to 3 minutes for CI checks and auto-reviewers (Copilot, Gemini, CodeRabbit) to post:

```bash
echo "Waiting for CI and auto-reviewers..."
for i in $(seq 1 18); do
  sleep 10
  STATUS=$(gh pr checks "$PR_NUM" --json name,state 2>/dev/null)
  PENDING=$(echo "$STATUS" | jq '[.[] | select(.state == "PENDING" or .state == "QUEUED")] | length')
  if [ "$PENDING" = "0" ]; then
    echo "All checks completed."
    break
  fi
  echo "  ... $PENDING checks still pending (${i}/18)"
done
```

## Step 3: Resolution Loop

For each iteration:

### 3a. Fetch CI Status

```bash
gh pr checks "$PR_NUM" --json name,state,conclusion
```

If any check failed, read the failure logs and attempt to fix:
```bash
gh run view {run_id} --log-failed 2>/dev/null | tail -50
```

### 3b. Fetch Unresolved Comments

```bash
# Get all review comments and review threads
gh api repos/{owner}/{repo}/pulls/{PR_NUM}/comments --paginate
gh api repos/{owner}/{repo}/pulls/{PR_NUM}/reviews --paginate
```

### 3c. Classify Each Comment

For each unresolved comment, classify it as one of:

| Type | Action |
|------|--------|
| **Code fix** | Read the referenced file, apply the fix, commit |
| **Style/nit** | Apply if trivial, skip with reply if subjective |
| **Question** | Reply with context from the codebase |
| **False positive** | Reply explaining why the current code is correct |
| **Out of scope** | Reply acknowledging, note for future work |

Use the `reviewer` agent to classify and draft responses:

```
Task: Classify and resolve this PR review comment.
Agent: reviewer
Context:
  - Comment: {comment_body}
  - File: {file_path}:{line}
  - Current code: (read the file)
  - Full PR diff context: git diff main...HEAD -- {file_path}

Classify as: code_fix | style_nit | question | false_positive | out_of_scope
If code_fix or style_nit: propose the exact change.
If question or false_positive: draft a reply.
```

### 3d. Apply Fixes and Reply

For code fixes:
```bash
# Apply the fix (Edit tool)
# Stage and commit
git add {files}
git commit -m "address review: {summary}"
```

For replies:
```bash
gh api repos/{owner}/{repo}/pulls/{PR_NUM}/comments/{comment_id}/replies \
  -f body="{reply}"
```

### 3e. Push and Request Re-Review

```bash
git push
# Request re-review from original reviewers if code was changed
gh pr edit "$PR_NUM" --add-reviewer {reviewers}
```

### 3f. Check Completion

```bash
REMAINING=$(gh api repos/{owner}/{repo}/pulls/{PR_NUM}/comments --paginate | \
  jq '[.[] | select(.resolved == false or .resolved == null)] | length')

CHECKS_OK=$(gh pr checks "$PR_NUM" --json conclusion | \
  jq '[.[] | select(.conclusion != "SUCCESS" and .conclusion != "NEUTRAL" and .conclusion != "SKIPPED")] | length')

if [ "$REMAINING" = "0" ] && [ "$CHECKS_OK" = "0" ]; then
  echo "All comments resolved and checks passing."
  break
fi
```

## Step 4: Report

```
## PR Monitor Report

**PR:** #{pr_num} — {title}
**Iterations:** {completed} / {max}
**Status:** {all_resolved | partial | stuck}

### CI Checks
| Check | Status | Notes |
|-------|--------|-------|
| build | ✓ pass | |
| test | ✓ pass | Fixed in iteration 2 |
| lint | ✓ pass | |

### Comments Resolved
| # | Type | File | Action | Iteration |
|---|------|------|--------|-----------|
| 1 | code_fix | src/auth.ts:42 | Fixed null check | 1 |
| 2 | question | src/config.ts:15 | Replied with context | 1 |
| 3 | false_positive | src/util.ts:88 | Explained rationale | 1 |
| 4 | style_nit | src/auth.ts:50 | Applied formatting | 2 |

### Unresolved (if any)
{list with reason why each couldn't be resolved}

### Commits Pushed
| Commit | Summary |
|--------|---------|
| abc1234 | address review: add null check in auth handler |
| def5678 | address review: formatting fixes |
```

## Rules

- Never force-push — always regular push
- Never resolve threads you didn't actually address
- Never dismiss reviews — only request re-review after fixing
- Classify before acting — don't blindly apply every suggestion
- Reply to false positives with evidence, not dismissal
- If a comment requires architectural changes, classify as out_of_scope and flag to user
- Stop after max iterations even if comments remain
- Use `AskUserQuestion` if a comment is ambiguous and classification is uncertain
- The reviewer agent runs in worktree isolation for classification
