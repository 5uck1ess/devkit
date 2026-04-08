# Stub & Placeholder Detection Patterns

Mechanical grep patterns for detecting unfinished work. Use in self-audit's Stale Code section or tri-review to flag incomplete implementations.

All matches require manual review — these are heuristics, not definitive indicators.

## TODO/FIXME Comments

```bash
grep -rn 'TODO\|FIXME\|HACK\|XXX\|PLACEHOLDER' \
  --include='*.go' --include='*.ts' --include='*.tsx' --include='*.js' --include='*.jsx' \
  --include='*.py' --include='*.rs' --include='*.rb' . 2>/dev/null
```

## Empty or Trivial Implementations

```bash
# Functions that return nothing useful — high false-positive rate.
# Filter results manually: return nil/null/{}/[] is often legitimate.
# Strongest signal when combined with TODO/FIXME nearby.
grep -rn 'return nil\|return null\|return undefined' \
  --include='*.go' --include='*.ts' --include='*.js' --include='*.py' . 2>/dev/null

# Python pass-only functions
grep -rn 'pass$' --include='*.py' . 2>/dev/null

# Empty catch/error blocks (use grep -E for extended regex portability)
grep -rEn 'catch[^{]*\{\s*\}' --include='*.ts' --include='*.tsx' --include='*.js' --include='*.jsx' . 2>/dev/null
```

## Placeholder Text

```bash
# UI placeholders left in
grep -rni 'lorem ipsum\|coming soon\|under construction\|placeholder' \
  --include='*.ts' --include='*.tsx' --include='*.js' --include='*.jsx' \
  --include='*.html' --include='*.vue' --include='*.svelte' . 2>/dev/null

# Template brackets left in
grep -rn '\[TODO\]\|<TODO>\|{TODO}' \
  --include='*.md' --include='*.ts' --include='*.tsx' --include='*.go' . 2>/dev/null
```

## Hardcoded Values Where Dynamic Expected

```bash
# Hardcoded IDs or secrets (not in test files)
grep -rn 'api_key\s*=\s*"[^"]\+"\|password\s*=\s*"[^"]\+"' \
  --include='*.go' --include='*.ts' --include='*.tsx' --include='*.py' . 2>/dev/null | grep -v '_test\.\|\.test\.\|\.spec\.'

# Hardcoded URLs (not config/const files)
grep -rn 'http://localhost\|127\.0\.0\.1' \
  --include='*.go' --include='*.ts' --include='*.tsx' --include='*.py' . 2>/dev/null | grep -v 'config\|const\|test\|spec'
```

## Disabled or Skipped Tests

```bash
# Skipped tests
grep -rn '\.Skip\|t\.Skip\|xit(\|xdescribe(\|@pytest\.mark\.skip\|@unittest\.skip' \
  --include='*.go' --include='*.ts' --include='*.js' --include='*.py' . 2>/dev/null

# Commented-out test blocks
grep -rn '// func Test\|// it(\|// test(\|# def test_' \
  --include='*.go' --include='*.ts' --include='*.js' --include='*.py' . 2>/dev/null
```
