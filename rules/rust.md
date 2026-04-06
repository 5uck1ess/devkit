---
globs: "*.rs"
---

# Rust Rules

Hooks already catch: unwrap-after-error, `let _ =` discard, unwrap on Option/Result in non-test code. These rules guide *how to write*.

- `?` for propagation. `thiserror` for libraries, `anyhow` for applications.
- Borrow (`&T`) over clone. `&str` in params, not `String`, unless ownership needed.
- `if let` / `let else` over `match` with one interesting arm.
- Iterators over manual loops. `enum` over boolean flags.
- Newtype for type safety: `struct UserId(u64)` not bare `u64`.
- `Debug` on all structs. `Clone`/`PartialEq` only when needed.
- `to_owned()` for `&str` → `String`. `to_string()` for Display types.
- Don't fight the borrow checker with `Rc<RefCell<T>>` unless genuinely needed.
- `cargo clippy` warnings are errors. `cargo fmt` for formatting.
