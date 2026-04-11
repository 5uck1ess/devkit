# Contributing to Devkit

## Adding a Workflow

Most command logic lives in YAML workflows executed by the Go engine. In current Claude Code all skills are tab-completable slash commands — new entry points go in `skills/`, not `commands/` (which is legacy).

1. Create `workflows/my-workflow.yml` with steps, model assignments, and loop/gate definitions
2. Test with `devkit_start` MCP tool or `devkit workflow my-workflow "input"` from terminal
3. Optionally add a context-activated skill in `skills/` to auto-trigger it

See `skills/creating-workflows/SKILL.md` for YAML schema reference.

### Adding a User-Only Entry Point

For side-effecting actions (install / deploy / setup) that should only run when the user explicitly invokes them, create a skill with `disable-model-invocation: true`:

1. Create `skills/my-skill/SKILL.md` with YAML frontmatter:
   ```markdown
   ---
   name: my-skill
   description: What this skill does and when to use it.
   disable-model-invocation: true
   ---
   # Skill Title
   Instructions...
   ```

This makes `/devkit:my-skill` tab-completable but prevents Claude from auto-triggering it — useful for installers (`setup-rules`) and other side-effecting operations. The `commands/` directory is legacy; don't add new files there.

## Adding a Context-Activated Skill

Skills activate automatically based on natural language — no slash command needed.

1. Create `skills/my-skill/SKILL.md` with YAML frontmatter:
   ```markdown
   ---
   name: my-skill
   description: Triggers on "natural language pattern".
   ---
   # Skill Title
   Guidelines or workflow (keep under 100 lines).
   ```

The skill will be auto-discovered as `devkit:my-skill`.

## Adding an Agent

1. Create `agents/my-agent.md` with YAML frontmatter specifying name, description, model, effort, maxTurns, and disallowedTools
2. Scope tools to only what the agent needs

## Adding a Workflow

1. Create a YAML file in `workflows/`
2. See the `creating-workflows` skill for schema reference
3. Run with `/my-workflow` — every workflow has its own tab-completable skill

## Guidelines

- Keep files concise — context space is expensive
- One logical change per PR
- Test commands manually before submitting
- Follow existing naming conventions (kebab-case)
- Don't duplicate built-in Claude Code commands

## Reporting Issues

Open an issue at https://github.com/5uck1ess/devkit/issues
