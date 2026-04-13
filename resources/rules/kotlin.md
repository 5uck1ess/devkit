---
paths:
  - "**/*.kt"
  - "**/*.kts"
---

# Kotlin Rules

- `val` over `var`. Immutable by default.
- Data classes for DTOs. Sealed classes/interfaces for closed hierarchies.
- `?.let { }` over null checks. `?:` (Elvis) for defaults. Avoid `!!` — it's a crash waiting to happen.
- `use { }` for `Closeable` resources (Kotlin's try-with-resources).
- Extension functions for utility — but only when they read like natural operations on the type.
- `when` over `if-else` chains. Exhaustive `when` on sealed types (no `else` branch needed).
- `listOf()` / `mapOf()` for read-only. `mutableListOf()` when mutation needed.
- Coroutines: `suspend` functions over callbacks. `withContext(Dispatchers.IO)` for blocking I/O.
- `require()` / `check()` for preconditions — they throw `IllegalArgumentException` / `IllegalStateException`.
- Named arguments when 2+ params of same type: `createUser(name = "x", email = "y")`.
