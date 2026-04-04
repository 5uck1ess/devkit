# Contributing to Devkit

## Adding a Slash Command (deterministic workflow)

Slash commands appear in tab-completion and run step-by-step workflows.

1. Create `skills/my-command.md` with YAML frontmatter:
   ```markdown
   ---
   name: devkit:my-command
   description: What this command does.
   ---
   # Command Title
   Step-by-step workflow with numbered steps.
   ```
2. Add `"skills/my-command.md"` to the `"commands"` array in `manifest.json`
3. Include Budget & Early Exit section if the command loops
4. Include `[PARALLEL]` markers if steps run concurrently

## Adding a Context-Activated Skill

Skills activate automatically based on natural language — no slash command needed.

1. Create `skills/my-skill.md` with YAML frontmatter:
   ```markdown
   ---
   name: devkit:my-skill
   description: Triggers on "natural language pattern".
   ---
   # Skill Title
   Guidelines or workflow (keep under 100 lines).
   ```
2. Add `"skills/my-skill.md"` to the `"skills"` array in `manifest.json`

## Adding an Agent

1. Create `agents/my-agent.md` with YAML frontmatter specifying model, tools, isolation, and maxTurns
2. Add `"agents/my-agent.md"` to `manifest.json`
3. Scope tools to only what the agent needs

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
