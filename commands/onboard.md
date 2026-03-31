---
name: devkit:onboard
description: Generate a codebase onboarding guide. Analyzes architecture, key files, patterns, and gotchas to help new contributors get up to speed.
---

# Codebase Onboarding

Analyze a codebase and generate an interactive onboarding guide for new contributors.

## Step 1: Analyze Structure

Read and analyze:
- Directory structure and organization
- Package manifests (package.json, go.mod, pyproject.toml, Cargo.toml, etc.)
- Entry points (main files, index files, cmd/ directories)
- Configuration files
- CI/CD setup
- README, CONTRIBUTING, CLAUDE.md if they exist

## Step 2: Identify Architecture

Spawn the `researcher` agent:

```
Task: Analyze the architecture of this codebase.
Agent: researcher
Context:
  - Root directory: {cwd}
  - Focus areas: entry points, data flow, key abstractions, external dependencies
```

The researcher should identify:
1. **Architecture pattern** (monolith, microservices, MVC, etc.)
2. **Key directories** and what lives in each
3. **Data flow** — how a request/event moves through the system
4. **Core abstractions** — the important types, interfaces, classes
5. **External dependencies** — APIs, databases, services
6. **Build and deploy** — how to build, test, and deploy

## Step 3: Generate Guide

```
## Onboarding: {project_name}

### Quick Start
1. Install dependencies: {install_command}
2. Run tests: {test_command}
3. Start dev server: {dev_command}

### Architecture
{architecture_summary}

### Directory Map
| Directory | Purpose |
|-----------|---------|
| src/api/  | REST API handlers |
| src/db/   | Database models and migrations |
| ...       | ... |

### Key Files
| File | Why it matters |
|------|---------------|
| src/server.ts | Entry point — starts HTTP server |
| src/middleware/auth.ts | Auth middleware — all routes go through this |
| ... | ... |

### Data Flow
{request_lifecycle_explanation}

### Patterns & Conventions
- {pattern_1}
- {pattern_2}

### Gotchas
- {gotcha_1}
- {gotcha_2}

### Common Tasks
| Task | How |
|------|-----|
| Add a new API endpoint | Create handler in src/api/, add route in src/routes.ts |
| Add a DB migration | ... |
| Run specific tests | ... |
```

## Presets

```
/devkit:onboard
/devkit:onboard --focus backend
/devkit:onboard --focus "authentication system"
```

## Rules

- Read actual code — don't guess from file names alone
- Focus on what a new contributor needs to be productive
- Keep it practical — commands, file paths, concrete examples
- Identify gotchas that aren't obvious from the code
- Skip boilerplate explanations (don't explain what node_modules is)
