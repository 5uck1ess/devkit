---
globs: "*.{ts,tsx,js,jsx,mjs,cjs}"
---

# TypeScript / JavaScript Rules

Hooks already catch: empty catch blocks, unhandled promises, `any` usage, security patterns (eval, innerHTML, etc). These rules guide *how to write*.

- `unknown` not `any`. Narrow with type guards.
- Discriminated unions over optional fields when exactly one variant applies.
- `catch (e: unknown)` and narrow — never `catch (e: any)`.
- `try/catch` at boundaries (API route, handler), not deep in logic.
- `const` over `let`. Never `var`. Named exports over default.
- `async/await` over `.then()` chains.
- Options object when 3+ params: `fn({ name, age, role })`.
- Early returns to flatten nesting. `??` over `||` when 0/"" are valid.
- `===` always. `JSON.parse` returns `any` — type immediately or validate (zod/valibot).
- `Array.sort()` mutates and sorts lexicographically — always pass comparator.
