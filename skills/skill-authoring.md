---
name: devkit:skill-authoring
description: How to write new skills for devkit — markdown format, frontmatter, progressive disclosure, keeping content concise.
---

# Skill Authoring

## Structure

A skill is a markdown file under `skills/`:

```
skills/
  my-skill.md              # Skill definition
```

## Format

Every skill file has two parts: YAML frontmatter and a markdown body.

```markdown
---
name: devkit:my-skill
description: One-line description of what this skill provides.
---

# Skill Title

Body content here.
```

### Frontmatter Fields

- **name** (required) — Identifier for the skill. Format: `devkit:kebab-case`.
- **description** (required) — Brief description. Shown in skill listings and used for matching, so make it specific.

## Writing the Body

The body is the instruction set loaded into context when the skill is activated.

**Keep it under 100 lines.** Context space is expensive. Every line should earn its place.

Guidelines:

- **Be direct.** State the principle, give one example, move on. Don't over-explain.
- **Use structure.** Headers, bullet points, and short code blocks are easier to follow than paragraphs.
- **Focus on decisions.** The most valuable guidance helps choose *between* options, not catalog all options.
- **Include examples** for anything where the format matters (schemas, naming conventions, file structure).
- **Omit the obvious.** Don't restate things any competent developer already knows.

## Registration

Add the skill to `manifest.json`:

```json
{
  "skills": [
    "skills/my-skill.md"
  ]
}
```

## Naming

- Use kebab-case: `writing-tests`, `clean-code`, `skill-authoring`
- Prefix with `devkit:` in the frontmatter name field
- Pick names that are natural to invoke: "use the clean-code skill"
