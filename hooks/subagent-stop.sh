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

# Evidence may be in the final message or in the Bash commands recorded in
# the subagent's own transcript (a concise summary often omits them). Only
# Bash tool_use commands count from the transcript — file contents and task
# text that merely mention "go test" are not evidence of running it.
EVIDENCE_TEXT="$AGENT_OUTPUT"
TX_COMMANDS=""
AGENT_TX=$(echo "$INPUT" | jq -r '.agent_transcript_path // empty' 2>/dev/null || true)
if [ -n "$AGENT_TX" ] && [ -f "$AGENT_TX" ] && [ ! -L "$AGENT_TX" ]; then
  TX_COMMANDS=$(tail -c 2000000 "$AGENT_TX" 2>/dev/null \
    | jq -rR 'fromjson? | .. | objects
        | select(.type == "tool_use" and .name == "Bash")
        | .input.command // empty' 2>/dev/null || true)
  EVIDENCE_TEXT="$EVIDENCE_TEXT
$TX_COMMANDS"
fi

# If there is no transcript evidence and the final message is empty or very
# short, block — something went wrong.
if [ -z "$TX_COMMANDS" ] && [ ${#AGENT_OUTPUT} -lt 20 ]; then
  jq -n '{
    decision: "block",
    reason: "Agent output is suspiciously short. Please verify your work is complete and run any test/metric commands before stopping."
  }'
  exit 0
fi

# Check for evidence that tests/metrics were actually run
TEST_EVIDENCE=false

# Go test
if grep -aqE '(PASS|FAIL|ok\s+\S+\s+[0-9.]+s|--- PASS|--- FAIL|go test)' <<< "$EVIDENCE_TEXT"; then
  TEST_EVIDENCE=true
fi

# Node/Jest/Vitest
if grep -aqE '(Tests?:\s+[0-9]|test suites?|✓|✗|✘|PASS\s|FAIL\s|npm test|npx jest|npx vitest)' <<< "$EVIDENCE_TEXT"; then
  TEST_EVIDENCE=true
fi

# Python pytest
if grep -aqE '(passed|failed|error).*(pytest|test)|pytest\s|python.*-m.*test' <<< "$EVIDENCE_TEXT"; then
  TEST_EVIDENCE=true
fi

# Go vet (race detector evidence)
if grep -aqE '(go vet|go test.*-race|-vet=|vet: )' <<< "$EVIDENCE_TEXT"; then
  TEST_EVIDENCE=true
fi

# Rust / linters
if grep -aqE '(cargo (test|clippy)|test result: ok|[0-9]+ passed|ruff check|eslint|golangci-lint|tsc --noEmit)' <<< "$EVIDENCE_TEXT"; then
  TEST_EVIDENCE=true
fi

# Generic pass/fail signals
if grep -aqE '(ALL_PASSING|ALL_DONE|ALL_TESTS_PASSING|BUILD_SUCCESS|LINT_CLEAN|RESEARCH_COMPLETE)' <<< "$EVIDENCE_TEXT"; then
  TEST_EVIDENCE=true
fi

# Metric command output (exit code references)
if grep -aqE '(exit\s+(code\s+)?0|metric.*pass|tests?\s+pass)' <<< "$EVIDENCE_TEXT"; then
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
