---
paths:
  - "**/*.sh"
---

# Shell Rules

Hooks already catch: macOS portability (grep -P, sed -i, readlink -f, stat --format, date -d, timeout, xargs -d). These rules guide *how to write*.

- `set -euo pipefail` always. No exceptions.
- Quote everything: `"$var"` not `$var`. `[[ ]]` not `[ ]`.
- `$(command)` not backticks. `printf` over `echo`.
- `local` for function vars. `readonly` for constants.
- `mktemp` + `trap 'rm -f "$tmpfile"' EXIT` for temp files.
- `command -v` not `which`. `${var:-default}` for defaults.
- `cd` changes dir permanently — use subshell: `(cd dir && command)`.
- Pipes create subshells — variables set in `while read | pipe` are lost.
- macOS: `sed -i ''`, `grep -E`, no `readlink -f`/`timeout`/`xargs -d`.
