# Contributing to Devkit

## Adding a Slash Command (deterministic workflow)

Slash commands appear in tab-completion and run step-by-step workflows.

1. Create `commands/my-command.md` with YAML frontmatter:
   ```markdown
   ---
   description: What this command does.
   ---
   # Command Title
   Step-by-step workflow with numbered steps.
   ```
2. Include Budget & Early Exit section if the command loops
3. Include `[PARALLEL]` markers if steps run concurrently

The command will be auto-discovered as `/devkit:my-command`.

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
