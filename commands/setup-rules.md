---
description: Install devkit coding rules to ~/.claude/rules/ for language-specific auto-activation
---

# Setup Coding Rules

Install language-specific coding rules to `~/.claude/rules/`. These auto-activate when Claude reads matching files and complement devkit's hooks — rules guide *how to write*, hooks catch *what you missed*.

## What it does

Copies rule files from this plugin's `resources/rules/` to `~/.claude/rules/`. Existing files are overwritten — if the user has customized rules, warn them before overwriting. `CLAUDE_PLUGIN_ROOT` is set automatically by the Claude Code plugin runtime.

## Steps

1. Create the rules directory:

```bash
mkdir -p ~/.claude/rules
```

2. Copy each rule file from the plugin:

```bash
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/go.md" ~/.claude/rules/go.md
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/typescript.md" ~/.claude/rules/typescript.md
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/python.md" ~/.claude/rules/python.md
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/rust.md" ~/.claude/rules/rust.md
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/shell.md" ~/.claude/rules/shell.md
```

3. Confirm installation:

```bash
echo "Installed rules:" && ls ~/.claude/rules/*.md
```

Report which files were installed.
