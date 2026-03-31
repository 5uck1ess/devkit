---
name: devkit:doc-gen
description: Generate documentation for target code. Analyzes exports, API surface, and usage patterns to produce structured docs.
---

# Documentation Generation

Analyze code and generate comprehensive documentation.

## Step 1: Analyze Target

Read the target files and identify:
- Public exports, classes, functions, types
- API surface and interfaces
- Configuration options
- Dependencies and relationships
- Usage patterns from existing code/tests

## Step 2: Generate Documentation

Spawn the `documenter` agent:

```
Task: Generate documentation for {target}.
Agent: documenter
Context:
  - Target: {target}
  - Doc format: {format or "markdown"}
  - Existing docs: {existing_doc_files}
  - User instructions: {args}
```

The documenter should produce:
1. **Overview** — what the module/package does
2. **API Reference** — every export with signature, params, return type, description
3. **Usage Examples** — realistic code snippets
4. **Configuration** — options and defaults if applicable

## Step 3: Output

Write docs to the appropriate location:
- If `docs/` directory exists, write there
- If a specific output path was requested, use that
- Otherwise, output inline in the conversation

## Presets

```
/devkit:doc-gen src/api/
/devkit:doc-gen lib/parser.go --format jsdoc
/devkit:doc-gen src/ --api-reference-only
```

## Rules

- Read actual code — don't guess signatures or behavior
- Include real examples, not placeholder code
- Match existing doc style if docs already exist
- Don't generate docs for internal/private code unless asked
- Keep descriptions concise — one line per param, one paragraph per function
