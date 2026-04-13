---
paths:
  - "**/*.java"
---

# Java Rules

- `Optional` return types, never null. `Optional.empty()` over `Optional.ofNullable(null)`.
- Records for immutable data carriers. Sealed interfaces for closed type hierarchies.
- `try-with-resources` for all `AutoCloseable`. Never manual `close()` in `finally`.
- `List.of()` / `Map.of()` for unmodifiable collections. `new ArrayList<>(List.of(...))` when mutation needed.
- `Objects.requireNonNull()` at public API boundaries with descriptive message.
- Stream pipelines for transforms. `for` loops for side effects or early exit.
- `private final` fields. Constructor injection over field injection.
- `BigDecimal` for money. Never `float`/`double` for currency.
- `@Override` always. Compiler catches signature drift.
- Checked exceptions for recoverable conditions. Runtime exceptions for programming errors.
