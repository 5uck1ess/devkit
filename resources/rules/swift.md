---
paths:
  - "**/*.swift"
---

# Swift Rules

- `let` over `var`. Immutable by default.
- `guard let` for early exit. `if let` for optional binding in the happy path.
- `struct` over `class` unless reference semantics are needed.
- `enum` with associated values over stringly-typed APIs.
- `throws` for recoverable errors. `fatalError()` only for truly impossible states.
- `[weak self]` in escaping closures that outlive the caller. `[unowned self]` only when lifetime is guaranteed.
- `async/await` over completion handlers. `Task { }` at boundaries, `await` inside.
- Access control: `private` by default, widen only as needed. `internal` is implicit — spell it out if intentional.
- `Codable` for serialization. Custom `init(from:)` only when the JSON shape differs from the model.
- Collections: prefer `map`/`filter`/`compactMap` over manual loops. `forEach` only for side effects.
