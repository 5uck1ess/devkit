# Contributing to Devkit

## Adding a Workflow

Most command logic lives in YAML workflows executed by the Go engine. Only 8 slash commands remain as tab-completable entry points.

1. Create `workflows/my-workflow.yml` with steps, model assignments, and loop/gate definitions
2. Test with `devkit workflow run my-workflow "input"`
3. Optionally add a context-activated skill in `skills/` to auto-trigger it

See `skills/creating-workflows/SKILL.md` for YAML schema reference.

### Adding a Slash Command (rare — only for top-level entry points)

Only add a command if it needs tab-completion. Most workflows are invoked via `devkit workflow run` or context-activated skills.

1. Create `commands/my-command.md` with YAML frontmatter:
   ```markdown
   ---
   description: What this command does.
   ---
   Run `devkit workflow run my-workflow` to execute.
   ```

The command name is derived from the filename: `commands/my-command.md` becomes `/devkit:my-command`.

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
3. Run with `/devkit:workflow my-workflow`

## Guidelines

- Keep files concise — context space is expensive
- One logical change per PR
- Test commands manually before submitting
- Follow existing naming conventions (kebab-case)
- Don't duplicate built-in Claude Code commands

## Reporting Issues

Open an issue at https://github.com/5uck1ess/devkit/issues
