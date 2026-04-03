#!/bin/bash
# devkit SubagentStop hook — prevents agents from exiting without running tests
#
# Checks the agent's transcript for evidence that a test/metric command was
# actually executed before allowing the agent to stop. This prevents the
# common failure mode where an agent claims "done" without verifying.
#
# Hook input (JSON on stdin):
#   .tool_name       = "Stop"
#   .agent_name      = the sub-agent's name
#   .agent_output    = the agent's final output text
#
# Exit codes:
#   0 + permissionDecision "allow"  → agent may stop
#   0 + permissionDecision "block"  → agent must continue (with reason)
#   1                               → hook error (allow by default)

set -euo pipefail

INPUT=$(cat)
AGENT_OUTPUT=$(echo "$INPUT" | jq -r '.agent_output // empty')

# If agent output is empty or very short, block — something went wrong
if [ ${#AGENT_OUTPUT} -lt 20 ]; then
  jq -n '{
    hookSpecificOutput: {
      hookEventName: "SubagentStop",
      permissionDecision: "block",
      permissionDecisionReason: "Agent output is suspiciously short. Please verify your work is complete and run any test/metric commands before stopping."
    }
  }'
  exit 0
fi

# Check for evidence that tests/metrics were actually run
# Look for common test runner output patterns
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

# Generic pass/fail signals
if echo "$AGENT_OUTPUT" | grep -qE '(ALL_PASSING|ALL_DONE|ALL_TESTS_PASSING|BUILD_SUCCESS|LINT_CLEAN|RESEARCH_COMPLETE)'; then
  TEST_EVIDENCE=true
fi

# Metric command output (exit code references)
if echo "$AGENT_OUTPUT" | grep -qE '(exit\s+(code\s+)?0|metric.*pass|tests?\s+pass)'; then
  TEST_EVIDENCE=true
fi

# If no test evidence found, block and ask the agent to verify
if [ "$TEST_EVIDENCE" = "false" ]; then
  jq -n '{
    hookSpecificOutput: {
      hookEventName: "SubagentStop",
      permissionDecision: "block",
      permissionDecisionReason: "No evidence of test/metric execution found in output. Run the test or metric command to verify your changes before stopping."
    }
  }'
  exit 0
fi

# All clear — agent ran tests
jq -n '{
  hookSpecificOutput: {
    hookEventName: "SubagentStop",
    permissionDecision: "allow"
  }
}'
exit 0
