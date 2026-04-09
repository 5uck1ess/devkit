---
description: Build an AST-based symbol index of the repository — exports, functions, classes, imports — cached for fast agent navigation.
---

# Repo Map

Build and cache an AST-based symbol index for fast codebase navigation.

## Steps

1. **Detect languages** — Scan for language markers (go.mod, package.json, pyproject.toml, Cargo.toml, etc.)
2. **Extract symbols (AST)** — Use language-specific AST tools to extract exports, functions, classes, types, interfaces. Fall back to regex if AST tools unavailable.
3. **Build dependency graph** — Map imports/requires between files to build a dependency graph
4. **Cache the map** — Write to `.devkit/repo-map.json` with current commit hash for staleness detection
5. **Generate report** — Summary: entry points, hub files (most imported), orphan files (never imported), symbol counts

## Rules

- Prefer AST extraction over regex — more accurate
- Cache results — don't rebuild on every invocation
- Include commit hash for staleness detection
- Identify entry points, hubs, and orphans
- Other commands (tri-review, refactor, decompose) can reference the cached map
