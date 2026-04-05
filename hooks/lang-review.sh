#!/bin/bash
# devkit PostToolUse hook — language-aware code quality review
#
# Consolidated hook that replaces go-review.sh and go-nil-return.sh.
# Detects language from file extension and runs the appropriate checks.
#
# Supported languages:
#   Go     — error-path result access, concurrent map access, nil-error returns, filepath traversal
#   TS/JS  — empty catch blocks, unhandled promise rejections, any-type usage
#   Rust   — unwrap-after-error, let _ = discard, unwrap on Option/Result in non-test code
#   Python — bare except, pass-in-except, mutable default arguments
#   Shell  — macOS portability (grep -P, sed -i, readlink -f, stat --format, etc.)
#
# PostToolUse hook schema:
#   { "hookSpecificOutput": { "hookEventName": "PostToolUse", "additionalContext": "string" } }

set -uo pipefail

# Safety net: if anything crashes, exit cleanly (hook must never die without output)
trap 'exit 0' ERR

INPUT=$(cat || true)
[ -z "$INPUT" ] && exit 0

TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty' 2>/dev/null || true)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null || true)
CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // .tool_input.new_string // empty' 2>/dev/null || true)

# Only check Edit/Write
if [ "$TOOL_NAME" != "Edit" ] && [ "$TOOL_NAME" != "Write" ]; then
  exit 0
fi
[ -z "$CONTENT" ] && exit 0

WARNINGS=""

add_warning() {
  WARNINGS="$WARNINGS\n- $1"
}

# ---------------------------------------------------------------------------
# Go (.go)
# ---------------------------------------------------------------------------
check_go() {
  # Error-path result access — simplified: check if any line between
  # "if err != nil {" and its closing "}" references result./res.
  if echo "$CONTENT" | grep -qE 'if err != nil'; then
    echo "$CONTENT" | awk '
      /if err != nil[[:space:]]*\{/ { in_err = 1; depth = 0 }
      in_err {
        for (i = 1; i <= length($0); i++) {
          c = substr($0, i, 1)
          if (c == "{") depth++
          if (c == "}") depth--
        }
        if (/result\.|res\./ && depth > 0) { matched = 1 }
        if (depth <= 0) { in_err = 0 }
      }
      END { exit matched ? 0 : 1 }
    ' && AWK_MATCHED=true || AWK_MATCHED=false
    if $AWK_MATCHED; then
      add_warning "Go: possible result field access inside error path (result may be zero-value when err != nil)"
    fi
  fi

  # Concurrent map access without protection
  if echo "$CONTENT" | grep -qE 'go func' && echo "$CONTENT" | grep -qE 'map\[string\]'; then
    if ! echo "$CONTENT" | grep -qE '(sync\.Mutex|sync\.RWMutex|sync\.Map|snapshot|Snap)'; then
      add_warning "Go: goroutines with map usage but no visible mutex/snapshot — verify concurrent map access is safe"
    fi
  fi

  # filepath.Join with unsanitized variable
  if echo "$CONTENT" | grep -qE 'filepath\.Join.*\b(name|input|arg|param|user)'; then
    if ! echo "$CONTENT" | grep -qE '(regexp|Regexp|MustCompile|MatchString|ValidateName|sanitize)'; then
      add_warning "Go: filepath.Join with potentially unsanitized input — validate before constructing paths"
    fi
  fi

  # Nil-error return detection (functions that always return nil error)
  NIL_FUNCS=$(echo "$CONTENT" | awk '
    /^func .*\)[[:space:]]*(\(.*error\)|error)[[:space:]]*\{/ {
      fname = $0; sub(/\{.*/, "", fname)
      in_func = 1; brace_depth = 0; has_return = 0; has_non_nil_err = 0
      line = $0
      for (j = 1; j <= length(line); j++) {
        c = substr(line, j, 1)
        if (c == "{") brace_depth++
        if (c == "}") brace_depth--
      }
      next
    }
    in_func {
      line = $0
      for (i = 1; i <= length(line); i++) {
        c = substr(line, i, 1)
        if (c == "{") brace_depth++
        if (c == "}") brace_depth--
      }
      if ($0 ~ /return /) {
        has_return = 1
        if ($0 !~ /,[[:space:]]*nil[[:space:]]*$/ && $0 !~ /return nil[[:space:]]*$/ && $0 !~ /,[[:space:]]*nil[[:space:]]*\)/) {
          has_non_nil_err = 1
        }
      }
      if (brace_depth <= 0) {
        if (has_return && !has_non_nil_err) {
          gsub(/^[[:space:]]+/, "", fname)
          print fname
        }
        in_func = 0
      }
    }
  ')
  if [ -n "$NIL_FUNCS" ]; then
    COUNT=$(echo "$NIL_FUNCS" | wc -l | tr -d ' ')
    FIRST=$(echo "$NIL_FUNCS" | head -1)
    add_warning "Go: ${COUNT} function(s) return error but only ever return nil (e.g., ${FIRST})"
  fi
}

# ---------------------------------------------------------------------------
# TypeScript / JavaScript (.ts, .tsx, .js, .jsx, .mjs, .cjs)
# ---------------------------------------------------------------------------
check_typescript() {
  # Empty catch blocks
  if echo "$CONTENT" | grep -qE 'catch\s*\([^)]*\)\s*\{\s*\}'; then
    add_warning "TS/JS: empty catch block swallows errors silently — log or rethrow"
  fi

  # catch with only console.log (no rethrow)
  if echo "$CONTENT" | grep -qE 'catch\s*\(' && echo "$CONTENT" | grep -qE 'console\.(log|warn)\('; then
    if ! echo "$CONTENT" | grep -qE '(throw|reject|process\.exit)'; then
      add_warning "TS/JS: catch block logs but doesn't rethrow — error may be silently swallowed"
    fi
  fi

  # any type usage
  if echo "$CONTENT" | grep -qE ':\s*any\b|<any>|as any'; then
    add_warning "TS: 'any' type usage — consider a specific type or 'unknown'"
  fi

  # Unhandled promise (async function without try/catch or .catch)
  if echo "$CONTENT" | grep -qE 'async\s+function|async\s*\('; then
    if ! echo "$CONTENT" | grep -qE '(try\s*\{|\.catch\(|await.*\.catch)'; then
      add_warning "TS/JS: async function without visible error handling — add try/catch or .catch()"
    fi
  fi
}

# ---------------------------------------------------------------------------
# Rust (.rs)
# ---------------------------------------------------------------------------
check_rust() {
  # .unwrap() in non-test code
  if echo "$CONTENT" | grep -qE '\.unwrap\(\)'; then
    if ! echo "$CONTENT" | grep -qE '(#\[test\]|#\[cfg\(test\)\]|mod tests)'; then
      add_warning "Rust: .unwrap() in non-test code — use ? operator or handle the error"
    fi
  fi

  # let _ = discarding Result/Option
  if echo "$CONTENT" | grep -qE 'let\s+_\s*=.*\b(Result|Option|Ok|Err)\b|let\s+_\s*=.*\?'; then
    add_warning "Rust: discarding Result/Option with let _ = — handle or explicitly document why"
  fi

  # expect() with non-descriptive message
  if echo "$CONTENT" | grep -qE '\.expect\(\s*"[^"]{0,10}"\s*\)'; then
    add_warning "Rust: .expect() with short message — provide a descriptive panic message"
  fi

  # unsafe block
  if echo "$CONTENT" | grep -qE '\bunsafe\s*\{'; then
    add_warning "Rust: unsafe block — verify memory safety invariants are maintained"
  fi
}

# ---------------------------------------------------------------------------
# Python (.py)
# ---------------------------------------------------------------------------
check_python() {
  # Bare except
  if echo "$CONTENT" | grep -qE '^\s*except\s*:'; then
    add_warning "Python: bare 'except:' catches everything including KeyboardInterrupt — use 'except Exception:'"
  fi

  # except with pass (silent swallow)
  if echo "$CONTENT" | grep -qE 'except.*:\s*$' && echo "$CONTENT" | grep -qE '^\s*pass\s*$'; then
    add_warning "Python: 'except: pass' silently swallows errors — log or handle"
  fi

  # Mutable default arguments
  if echo "$CONTENT" | grep -qE 'def\s+\w+\(.*=\s*(\[\]|\{\}|set\(\))'; then
    add_warning "Python: mutable default argument (list/dict/set) — use None and initialize inside function"
  fi

  # Generic Exception catch
  if echo "$CONTENT" | grep -qE 'except\s+Exception\s+as\s+\w+\s*:\s*$' && echo "$CONTENT" | grep -qE '^\s*pass\s*$'; then
    add_warning "Python: catching Exception and passing — error is silently lost"
  fi
}

# ---------------------------------------------------------------------------
# Shell (.sh)
# ---------------------------------------------------------------------------
check_shell() {
  # macOS portability checks
  echo "$CONTENT" | grep -qE 'grep\s+(-[a-zA-Z]*P|--perl-regexp)' && \
    add_warning "Shell: grep -P (Perl regex) unavailable on macOS — use grep -E, awk, or perl"

  echo "$CONTENT" | grep -qE 'sed\s+-i\s+[^'"'"'"]' && \
    add_warning "Shell: sed -i without '' breaks on macOS BSD sed — use sed -i ''"

  echo "$CONTENT" | grep -qE 'readlink\s+-f\b' && \
    add_warning "Shell: readlink -f is GNU-only — use realpath or manual loop on macOS"

  echo "$CONTENT" | grep -qE 'stat\s+--format' && \
    add_warning "Shell: stat --format is GNU-only — use stat -f on macOS"

  echo "$CONTENT" | grep -qE 'xargs\s+-d\b' && \
    add_warning "Shell: xargs -d is GNU-only — use tr + xargs on macOS"

  echo "$CONTENT" | grep -qE 'date\s+-d\b' && \
    add_warning "Shell: date -d is GNU-only — use date -j -f on macOS"

  echo "$CONTENT" | grep -qE '\btimeout\s+[0-9]' && \
    add_warning "Shell: timeout command is GNU-only — use perl -e 'alarm N; exec @ARGV' on macOS"
}

# ---------------------------------------------------------------------------
# Dispatch by file extension
# ---------------------------------------------------------------------------
case "$FILE_PATH" in
  *.go)                          check_go ;;
  *.ts|*.tsx|*.js|*.jsx|*.mjs|*.cjs) check_typescript ;;
  *.rs)                          check_rust ;;
  *.py)                          check_python ;;
  *.sh)                          check_shell ;;
  *)                             exit 0 ;;
esac

if [ -n "$WARNINGS" ]; then
  MSG=$(printf "Code quality check:%b" "$WARNINGS")
  jq -n --arg msg "$MSG" '{
    hookSpecificOutput: {
      hookEventName: "PostToolUse",
      additionalContext: $msg
    }
  }'
  exit 0
fi

exit 0
