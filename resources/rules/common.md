---
paths:
  - "**/*"
---

# Common Rules

Language-agnostic principles. Applied alongside language-specific rules.

- State assumptions before implementing. If multiple interpretations exist, present them — don't pick silently.
- One concern per commit. One concern per function. If you say "and", split it.
- Name things for what they do, not where they came from or how they work.
- Delete dead code. Commented-out code is dead code.
- Tests prove behavior, not implementation. If the test breaks on a refactor, it tested the wrong thing.
- Error messages include: what happened, what was expected, what to do next.
- Validate at system boundaries (user input, external APIs, file I/O). Trust internal code.
- Paths: use forward slashes or `path.join`/`filepath.Join` — never hardcode `\`.
- Line endings: let `.gitattributes` or the runtime handle it — never assume `\n`.
- File operations: use `os.MkdirAll`/`makedirs(exist_ok=True)` — never assume dirs exist.
- Temp files: use the OS temp directory (`os.TempDir()`/`tempfile`/`os.tmpdir()`) — never hardcode `/tmp`.
