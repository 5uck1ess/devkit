# Contributing to Devkit

## Adding a Skill

1. Create `skills/my-skill.md` with YAML frontmatter:
   ```markdown
   ---
   name: devkit:my-skill
   description: One-line description.
   ---
   # Skill Title
   Body content (keep under 100 lines).
   ```
2. Add `"skills/my-skill.md"` to `manifest.json`
3. See the "Adding an Invocable Skill" section below for full guidance

## Adding an Invocable Skill (formerly "command")

1. Create `skills/my-skill.md` with YAML frontmatter:
   ```markdown
   ---
   name: devkit:my-skill
   description: What this skill does.
   ---
   # Skill Title
   Step-by-step workflow.
   ```
2. Add `"skills/my-skill.md"` to `manifest.json`
3. Include Budget & Early Exit section if the skill loops
4. Include `[PARALLEL]` markers if steps run concurrently

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
