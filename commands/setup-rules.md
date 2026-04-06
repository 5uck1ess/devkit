---
description: Install devkit coding rules to ~/.claude/rules/ for language-specific auto-activation
---

# Setup Coding Rules

Install language-specific coding rules to `~/.claude/rules/`. These auto-activate when Claude reads matching files and complement devkit's hooks — rules guide *how to write*, hooks catch *what you missed*.

## What it does

Copies rule files from this plugin's `resources/rules/` to `~/.claude/rules/`. Existing files are overwritten — if the user has customized rules, warn them before overwriting. `CLAUDE_PLUGIN_ROOT` is set automatically by the Claude Code plugin runtime.

## Steps

1. Verify plugin context and create the rules directory:

```bash
if [ -z "${CLAUDE_PLUGIN_ROOT:-}" ]; then
  echo "ERROR: CLAUDE_PLUGIN_ROOT is not set. Run this as /devkit:setup-rules." >&2
  exit 1
fi
mkdir -p ~/.claude/rules
```

2. Check for existing customized rules — warn before overwriting:

If any files in `~/.claude/rules/` differ from the plugin versions, tell the user which files will be overwritten and ask for confirmation before proceeding.

3. Copy each rule file from the plugin:

```bash
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/go.md" ~/.claude/rules/go.md
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/typescript.md" ~/.claude/rules/typescript.md
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/python.md" ~/.claude/rules/python.md
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/rust.md" ~/.claude/rules/rust.md
cp "${CLAUDE_PLUGIN_ROOT}/resources/rules/shell.md" ~/.claude/rules/shell.md
```

4. Confirm installation:

```bash
echo "Installed rules:" && ls ~/.claude/rules/*.md
```

Report which files were installed.
