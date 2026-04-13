---
paths:
  - "**/*.cs"
---

# C# Rules

- `readonly` fields, `init` properties. Immutable where possible.
- Records for immutable data. `record struct` when value semantics + small size.
- `using` declaration (not block) for `IDisposable`. `await using` for `IAsyncDisposable`.
- `??` for null coalescing. `?.` for null conditional. Avoid `!` (null-forgiving) — fix the nullability instead.
- Pattern matching: `is`, `switch` expressions, relational/logical patterns over chains of `if`.
- `async Task` over `async void` (except event handlers). Never `.Result` or `.Wait()` — deadlock risk.
- `IReadOnlyList<T>` / `IReadOnlyDictionary<K,V>` in public APIs. Mutable types stay internal.
- Dependency injection via constructor. `IOptions<T>` for configuration.
- `string.Equals(a, b, StringComparison.Ordinal)` for case-sensitive. `OrdinalIgnoreCase` for insensitive.
- `Path.Combine()` for file paths. `Environment.GetFolderPath()` for special directories.
