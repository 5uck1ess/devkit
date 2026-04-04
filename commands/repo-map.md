---
name: devkit:repo-map
description: Build an AST-based symbol index of the repository — exports, functions, classes, imports — cached for fast agent navigation.
---

# Repo Map

Builds a structural map of the codebase using AST parsing. The map is cached and used by other commands and agents for faster, more accurate navigation.

## Parameters

1. **Target** — directory to map (default: project root)
2. **Format** — output format: `summary` (default), `full`, `json`

## Prerequisites

Requires [ast-grep](https://ast-grep.github.io/) (`sg`) for AST parsing. Falls back to regex-based extraction if not installed.

```bash
# Check for ast-grep
if command -v sg >/dev/null 2>&1; then
  echo "ast-grep: $(sg --version)"
  MODE="ast"
else
  echo "ast-grep not installed — using regex fallback"
  echo "Install: brew install ast-grep  OR  npm i -g @ast-grep/cli"
  MODE="regex"
fi
```

## Step 1: Detect Languages

```bash
echo "=== Language Detection ==="
[ -n "$(find . -name '*.ts' -o -name '*.tsx' | head -1)" ] && echo "typescript"
[ -n "$(find . -name '*.js' -o -name '*.jsx' | head -1)" ] && echo "javascript"
[ -n "$(find . -name '*.py' | head -1)" ] && echo "python"
[ -n "$(find . -name '*.go' | head -1)" ] && echo "go"
[ -n "$(find . -name '*.rs' | head -1)" ] && echo "rust"
[ -n "$(find . -name '*.java' | head -1)" ] && echo "java"
[ -n "$(find . -name '*.rb' | head -1)" ] && echo "ruby"
```

## Step 2: Extract Symbols (AST mode)

For each detected language, use `sg` to extract structural elements:

### TypeScript / JavaScript

```bash
# Exported functions
sg --pattern 'export function $NAME($$$PARAMS): $RET { $$$ }' --lang ts -j
sg --pattern 'export const $NAME = ($$$PARAMS) => $$$' --lang ts -j
sg --pattern 'export default function $NAME($$$) { $$$ }' --lang ts -j

# Exported classes
sg --pattern 'export class $NAME $$${ $$$ }' --lang ts -j

# Exported interfaces/types
sg --pattern 'export interface $NAME { $$$ }' --lang ts -j
sg --pattern 'export type $NAME = $$$' --lang ts -j

# Imports (to build dependency graph)
sg --pattern 'import { $$$ } from "$SOURCE"' --lang ts -j
sg --pattern 'import $NAME from "$SOURCE"' --lang ts -j
```

### Python

```bash
# Functions
sg --pattern 'def $NAME($$$):' --lang python -j

# Classes
sg --pattern 'class $NAME($$$):' --lang python -j
sg --pattern 'class $NAME:' --lang python -j

# Imports
sg --pattern 'from $MODULE import $$$' --lang python -j
sg --pattern 'import $MODULE' --lang python -j
```

### Go

```bash
# Exported functions (capitalized)
sg --pattern 'func $NAME($$$) $$${ $$$ }' --lang go -j

# Structs
sg --pattern 'type $NAME struct { $$$ }' --lang go -j

# Interfaces
sg --pattern 'type $NAME interface { $$$ }' --lang go -j

# Imports
sg --pattern 'import "$PKG"' --lang go -j
```

### Rust

```bash
# Public functions
sg --pattern 'pub fn $NAME($$$) -> $$$ { $$$ }' --lang rust -j
sg --pattern 'pub fn $NAME($$$) { $$$ }' --lang rust -j

# Structs
sg --pattern 'pub struct $NAME { $$$ }' --lang rust -j

# Traits
sg --pattern 'pub trait $NAME { $$$ }' --lang rust -j

# Impls
sg --pattern 'impl $NAME { $$$ }' --lang rust -j
```

## Step 3: Extract Symbols (Regex fallback)

If `sg` is not installed, use grep-based extraction:

```bash
# Functions
grep -rnE '^\s*(export\s+)?(async\s+)?function\s+\w+' --include='*.ts' --include='*.js' | head -200
grep -rnE '^\s*def\s+\w+' --include='*.py' | head -200
grep -rnE '^func\s+\w+' --include='*.go' | head -200

# Classes
grep -rnE '^\s*(export\s+)?class\s+\w+' --include='*.ts' --include='*.js' | head -200
grep -rnE '^class\s+\w+' --include='*.py' | head -200

# Types/Interfaces
grep -rnE '^\s*(export\s+)?(interface|type)\s+\w+' --include='*.ts' | head -200
grep -rnE '^type\s+\w+\s+(struct|interface)' --include='*.go' | head -200
```

## Step 4: Build Dependency Graph

From the import data, build a file-level dependency graph:

```
src/auth/handler.ts
  → imports from: src/auth/middleware.ts, src/db/users.ts, src/config.ts
  ← imported by: src/routes/api.ts

src/db/users.ts
  → imports from: src/db/connection.ts, src/types.ts
  ← imported by: src/auth/handler.ts, src/admin/users.ts
```

Identify:
- **Entry points** — files with no importers (likely main/index files)
- **Hubs** — files imported by 5+ other files (high-impact change targets)
- **Orphans** — files that are never imported (possibly dead code)

## Step 5: Cache the Map

Write the map to `.devkit/repo-map.json`:

```json
{
  "generated": "2026-04-04T12:00:00Z",
  "commit": "abc1234",
  "languages": ["typescript", "python"],
  "mode": "ast",
  "symbols": {
    "src/auth/handler.ts": {
      "exports": [
        { "name": "handleLogin", "type": "function", "line": 15 },
        { "name": "handleLogout", "type": "function", "line": 42 },
        { "name": "AuthConfig", "type": "interface", "line": 5 }
      ],
      "imports": ["src/auth/middleware", "src/db/users", "src/config"]
    }
  },
  "graph": {
    "entry_points": ["src/index.ts", "src/cli.ts"],
    "hubs": ["src/config.ts", "src/types.ts", "src/db/connection.ts"],
    "orphans": ["src/legacy/old-parser.ts"]
  },
  "stats": {
    "files": 47,
    "functions": 128,
    "classes": 12,
    "interfaces": 23,
    "total_exports": 163
  }
}
```

The cache is invalidated when the current commit differs from the stored commit.

## Step 6: Generate Report

### Summary format (default)

```
## Repo Map

**Languages:** TypeScript, Python
**Mode:** AST (ast-grep)
**Files:** 47 | **Functions:** 128 | **Classes:** 12 | **Interfaces:** 23

### Entry Points
- src/index.ts (main application entry)
- src/cli.ts (CLI entry)

### Hubs (high-impact files)
- src/config.ts — imported by 12 files
- src/types.ts — imported by 9 files
- src/db/connection.ts — imported by 7 files

### Orphans (possibly dead code)
- src/legacy/old-parser.ts — never imported

### Top-Level Structure
src/
  auth/     — 4 files, 8 exports (handleLogin, handleLogout, ...)
  db/       — 3 files, 6 exports (getUser, createUser, ...)
  routes/   — 5 files, 10 exports (apiRouter, authRouter, ...)
  utils/    — 2 files, 4 exports (parseDate, formatCurrency, ...)

Cached to .devkit/repo-map.json
```

## Usage by Other Commands

Other devkit commands and agents can reference the cached map:

```bash
# Check if map exists and is current
if [ -f .devkit/repo-map.json ]; then
  MAP_COMMIT=$(jq -r '.commit' .devkit/repo-map.json)
  CURRENT=$(git rev-parse --short HEAD)
  if [ "$MAP_COMMIT" = "$CURRENT" ]; then
    echo "Using cached repo map"
    cat .devkit/repo-map.json
  else
    echo "Map is stale — rebuilding"
    # Trigger rebuild
  fi
fi
```

Commands that benefit from the map:
- **tri-review** — knows which files are hubs and need more careful review
- **refactor** — sees the dependency graph to understand blast radius
- **decompose** — uses the graph to identify natural task boundaries
- **onboard** — uses entry points and hubs to guide the tour
- **audit** — identifies orphaned/dead code

## Rules

- Always respect .gitignore — don't index node_modules, dist, etc.
- Cache to `.devkit/repo-map.json` (already gitignored)
- Invalidate cache on commit change
- Prefer AST mode — fall back to regex only when sg is not installed
- Cap extraction at 200 results per query to avoid context bloat
- The map is read-only — it never modifies code
- Skip binary files and files larger than 100KB
