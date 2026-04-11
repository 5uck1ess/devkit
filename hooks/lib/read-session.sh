#!/usr/bin/env bash
# read-session.sh: shared session.json parser sourced by devkit-guard.sh
# and devkit-stop-guard.sh. NOT executed directly.
#
# parse_session_fields <session-file>
#   On success sets: SESSION_STATUS, SESSION_STEP_TYPE, SESSION_ENFORCE,
#   SESSION_CURRENT_STEP, SESSION_WORKFLOW, SESSION_CURRENT_INDEX,
#   SESSION_TOTAL_STEPS, SESSION_STALE ("1" if older than TTL, "0"
#   otherwise). Returns 0.
#   On parse failure returns 1 (caller decides fail-open vs fail-closed).
#   On missing file sets SESSION_STATUS="" and returns 0.
#
# Stale check reads updated_at (ISO 8601 from Go time.Time); if missing
# or unparseable the session is treated as fresh. Falls back to
# started_at for backward compatibility with pre-UpdatedAt state files.

# 30 minutes — mirrors sessionStaleTTL in src/mcp/tools.go.
: "${DEVKIT_SESSION_STALE_TTL_SECONDS:=1800}"

parse_session_fields() {
  local session_file="$1"

  SESSION_STATUS=""
  SESSION_STEP_TYPE=""
  SESSION_ENFORCE=""
  SESSION_CURRENT_STEP=""
  SESSION_WORKFLOW=""
  SESSION_CURRENT_INDEX=""
  SESSION_TOTAL_STEPS=""
  SESSION_STALE="0"

  if [[ ! -f "$session_file" ]]; then
    return 0
  fi

  # Single python3 call. Path passed via sys.argv to prevent shell
  # injection. FileNotFoundError is TOCTOU between the -f test and
  # open() — treat as "no session" to match the pre-check intent.
  local parsed
  parsed=$(python3 -c '
import json, sys, datetime
try:
    d = json.load(open(sys.argv[1]))
except FileNotFoundError:
    print("\t".join([""] * 8))
    sys.exit(0)

ttl = int(sys.argv[2])
stale = "0"
ts = d.get("updated_at") or d.get("started_at") or ""
if ts:
    # Go marshals time.Time as RFC3339 with nanoseconds, e.g.
    # "2026-04-10T12:34:56.789Z". fromisoformat handles Z only on
    # 3.11+, so strip trailing Z for older pythons.
    if ts.endswith("Z"):
        ts = ts[:-1] + "+00:00"
    try:
        t = datetime.datetime.fromisoformat(ts)
        age = (datetime.datetime.now(datetime.timezone.utc) - t).total_seconds()
        if age > ttl:
            stale = "1"
    except ValueError:
        pass  # unparseable timestamp — treat as fresh, fail-safe

print("\t".join([
    d.get("status", ""),
    d.get("step_type", ""),
    d.get("enforce", "hard") or "hard",
    d.get("current_step", ""),
    d.get("workflow", ""),
    str(d.get("current_index", "")),
    str(d.get("total_steps", "")),
    stale,
]))
' "$session_file" "$DEVKIT_SESSION_STALE_TTL_SECONDS" 2>/dev/null) || return 1

  IFS=$'\t' read -r \
    SESSION_STATUS \
    SESSION_STEP_TYPE \
    SESSION_ENFORCE \
    SESSION_CURRENT_STEP \
    SESSION_WORKFLOW \
    SESSION_CURRENT_INDEX \
    SESSION_TOTAL_STEPS \
    SESSION_STALE \
    <<< "$parsed"

  return 0
}
