---
globs: "*.go"
---

# Go Rules

Hooks already catch: error-path result access, concurrent map without mutex, filepath traversal, nil-error returns, security patterns. These rules guide *how to write* — hooks catch *what you missed*.

- Wrap errors: `fmt.Errorf("doing X: %w", err)` — bare `return err` loses context.
- `context.Context` first param for I/O or blocking calls.
- `defer mu.Unlock()` immediately after `mu.Lock()`.
- Table-driven tests with `t.Run`. Use `t.Helper()` in test helpers.
- One package per directory. `main.go` stays thin: flags, wiring, `run()`.
- Imports: stdlib, blank, external, blank, internal.
- `defer` in loops defers until function exit — use closure or extract.
- `json.Unmarshal` into `interface{}` gives `float64` for numbers — use concrete types.
- `range` vars reused pre-1.22 — capture in closure for goroutines.
