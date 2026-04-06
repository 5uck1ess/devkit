---
globs: "*.py"
---

# Python Rules

Hooks already catch: bare except, pass-in-except, mutable defaults, security patterns (eval, exec, pickle, os.system, yaml.load). These rules guide *how to write*.

- `raise ... from err` to preserve exception chains.
- Type hints on function signatures. `X | None` over `Optional[X]`.
- `TypedDict` for dict shapes passed around, not bare `dict`.
- `dataclass` or `NamedTuple` over plain tuples/dicts for structured data.
- f-strings over `.format()`. `pathlib` over `os.path.join`.
- `with` for resource cleanup — files, locks, connections.
- Comprehensions over `map/filter` with lambdas. `enumerate()` over manual index.
- `is` only for `None`/`True`/`False`. `==` for value comparison.
- `ruff` for formatting and linting.
