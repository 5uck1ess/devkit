#!/bin/bash
# devkit SubagentStop hook — prevents code-modifying devkit agents from
# exiting without running tests.
#
# Only gates agents whose job is to change code and prove it works
# (test-writer, improver). Read-only agents (Explore, reviewer, researcher,
# security-auditor, documenter, third-party agents) are always approved —
# blocking them for lack of test output would be wrong.
#
# SubagentStop input (Claude Code): agent_type, last_assistant_message,
# agent_transcript_path, stop_hook_active, ...
# SubagentStop hook schema:
#   { "decision": "approve" | "block", "reason": "string" }

set -euo pipefail

# Must always emit a JSON verdict — jq failures degrade to empty values.
INPUT=$(cat || true)
AGENT_TYPE=$(echo "$INPUT" | jq -r '.agent_type // empty' 2>/dev/null || true)
STOP_HOOK_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active // false' 2>/dev/null || echo false)
AGENT_OUTPUT=$(echo "$INPUT" | jq -r '.last_assistant_message // empty' 2>/dev/null || true)

approve() { jq -n '{ decision: "approve" }'; exit 0; }

# Plugin agents may be namespaced (devkit:test-writer) — match the suffix.
case "${AGENT_TYPE##*:}" in
  test-writer|improver) ;;
  *) approve ;;
esac

# Already continuing because of an earlier block — don't loop.
[ "$STOP_HOOK_ACTIVE" = "true" ] && approve

# If agent output is empty or very short, block — something went wrong
if [ ${#AGENT_OUTPUT} -lt 20 ]; then
  jq -n '{
    decision: "block",
    reason: "Agent output is suspiciously short. Please verify your work is complete and run any test/metric commands before stopping."
  }'
  exit 0
fi

# Check for evidence that tests/metrics were actually run
TEST_EVIDENCE=false

# Go test
if echo "$AGENT_OUTPUT" | grep -qE '(PASS|FAIL|ok\s+\S+\s+[0-9.]+s|--- PASS|--- FAIL|go test)'; then
  TEST_EVIDENCE=true
fi

# Node/Jest/Vitest
if echo "$AGENT_OUTPUT" | grep -qE '(Tests?:\s+[0-9]|test suites?|✓|✗|✘|PASS\s|FAIL\s|npm test|npx jest|npx vitest)'; then
  TEST_EVIDENCE=true
fi

# Python pytest
if echo "$AGENT_OUTPUT" | grep -qE '(passed|failed|error).*(pytest|test)|pytest\s|python.*-m.*test'; then
  TEST_EVIDENCE=true
fi

# Go vet (race detector evidence)
if echo "$AGENT_OUTPUT" | grep -qE '(go vet|go test.*-race|-vet=|vet: )'; then
  TEST_EVIDENCE=true
fi

# Generic pass/fail signals
if echo "$AGENT_OUTPUT" | grep -qE '(ALL_PASSING|ALL_DONE|ALL_TESTS_PASSING|BUILD_SUCCESS|LINT_CLEAN|RESEARCH_COMPLETE)'; then
  TEST_EVIDENCE=true
fi

# Metric command output (exit code references)
if echo "$AGENT_OUTPUT" | grep -qE '(exit\s+(code\s+)?0|metric.*pass|tests?\s+pass)'; then
  TEST_EVIDENCE=true
fi

# If no test evidence found, block
if [ "$TEST_EVIDENCE" = "false" ]; then
  jq -n '{
    decision: "block",
    reason: "No evidence of test/metric execution found in output. Run the test or metric command to verify your changes before stopping."
  }'
  exit 0
fi

# All clear
jq -n '{
  decision: "approve"
}'
exit 0
