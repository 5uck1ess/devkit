---
name: dont-reinvent
description: Don't reinvent the wheel — use existing libraries, tools, and stdlib before building custom solutions. Every custom solution is maintenance burden.
---

# Don't Reinvent the Wheel

Before writing any non-trivial utility, helper, or tool — check if it already exists.

## The Search Order

1. **Language stdlib / builtins** — `path.join()`, `os.path`, `filepath.Join()` exist. Don't write your own.
2. **Framework built-ins** — most frameworks have solutions for routing, validation, auth, caching. Use them.
3. **Established packages** — if there's a well-maintained package with meaningful adoption and recent commits, prefer it over custom code.
4. **Existing tools / plugins / MCP servers** — before building a new CLI wrapper or integration, check if one exists. Run a quick search.
5. **Internal codebase** — search the project for existing utilities before creating new ones. Someone may have already solved this.

Only build custom when none of the above genuinely fit.

## Why This Matters

Every custom solution is code **you maintain forever**:
- Bug fixes are your problem
- Edge cases are your problem
- Updates when dependencies change are your problem
- Documentation is your problem

A well-maintained library handles all of this for you. The 20 minutes you "save" by writing it yourself costs hours in future maintenance.

## When Custom Is Justified

- The existing solutions don't fit and adapting them is harder than building
- The existing solutions are unmaintained (no commits in 12+ months, unresolved security issues)
- The scope is truly trivial (a 3-line helper doesn't need a dependency)
- Performance requirements rule out general-purpose solutions (measure first, don't assume)
- Security requirements demand full control over the implementation

## Red Flags You're Reinventing

- Writing a date parser, URL parser, retry wrapper, or logger from scratch
- Building a custom HTTP client wrapper around fetch/axios
- Writing string manipulation utilities that exist in lodash/underscore/stdlib
- Creating a custom config file format instead of using JSON/YAML/TOML
- Building a task queue when BullMQ/Celery/existing solutions exist
- Writing a custom test framework instead of using the ecosystem standard

## Evaluating Existing Solutions

Before adopting a dependency, check:

- **Last commit date** — no commits in 12+ months is a warning sign
- **Open issues** — many unresolved issues with no maintainer response suggests abandonment
- **Downloads / stars** — social proof that others rely on it
- **License compatibility** — GPL in an MIT project is a problem
- **Bundle size** — don't add a 500KB dependency for one function (use the function directly)

## The Maintenance Test

Before writing custom code, ask: "Am I willing to maintain this for the life of the project?" If the answer is no, find an existing solution. If no existing solution fits, keep the custom code as simple as possible — the less code you write, the less you maintain.
