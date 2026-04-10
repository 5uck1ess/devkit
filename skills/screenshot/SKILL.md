---
name: screenshot
description: Capture a screenshot or PDF of a web page — use when asked to screenshot a URL, capture a page, take a picture of a website, generate a visual snapshot, or save a page as PDF. Supports full page, element selector, device emulation, and custom viewport.
---

# Web Page Screenshot

Capture screenshots and PDFs of web pages using Playwright.

## Step 1: Verify Playwright

Run `npx playwright --version`. If it fails, stop and tell the user: `Playwright required — install with: npx playwright install chromium`. Do not attempt workarounds.

## Step 2: Parse and Validate

Extract from the user's request:

- **URL** — required; validated below
- **Output path** — default `./screenshot-{timestamp}.png` in cwd
- **Capture mode** — viewport (default), full-page, element selector
- **Device / viewport** — optional emulation or custom size
- **Format** — PNG (default) or PDF

**Validate the URL** (see Rules for the full list): only `http(s)`, no private/reserved IPs or cloud metadata, no `@`. On rejection, report the exact reason and stop — never silently skip in a batch.

## Step 3: Execute

Choose the command that matches the capture mode. Always quote the URL and output path.

**Viewport screenshot:**
```bash
npx playwright screenshot "{url}" "{output}"
```

**Full-page screenshot:**
```bash
npx playwright screenshot --full-page "{url}" "{output}"
```

**Custom viewport** (format is `W,H` with no spaces, e.g. `1280,800`):
```bash
npx playwright screenshot --viewport-size=1280,800 "{url}" "{output}"
```

**Device emulation:**
```bash
npx playwright screenshot --device="{device}" "{url}" "{output}"
```

**PDF output:**
```bash
npx playwright pdf "{url}" "{output}.pdf"
```

**Element selector** (Playwright CLI doesn't expose this directly, so write a script file and pass args via `process.argv` — never interpolate user strings into `node -e`):

```bash
cat > /tmp/devkit-shot-element.mjs <<'EOF'
import { chromium } from 'playwright';

const [, , url, selector, output] = process.argv;
if (!url || !selector || !output) {
  console.error('usage: node devkit-shot-element.mjs <url> <selector> <output>');
  process.exit(2);
}

const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 });
  await page.locator(selector).screenshot({ path: output });
  console.log(output);
} catch (err) {
  console.error('screenshot failed:', err.message);
  process.exit(1);
} finally {
  try { await browser.close(); } catch (e) { /* suppress close errors */ }
}
EOF

node /tmp/devkit-shot-element.mjs "$URL" "$SELECTOR" "$OUTPUT"
```

Pass all user values via shell variables. The `.mjs` reads them from `process.argv` so quoting, `$()`, and backticks in the input can't escape.

## Step 4: Report

Output the absolute path and any metadata:

```
Screenshot saved: /abs/path/to/file.png
Size: 342 KB
Mode: full-page
```

If the page failed to load, report the error clearly with the HTTP status or timeout reason.

## Rules

- **URL validation** — only `http://` and `https://`. Reject `file://`, `ftp://`, `data:`, and all other schemes. Reject private/reserved IPs: `localhost`, `127.0.0.1`, `0.0.0.0`, `10.x.x.x`, `172.16-31.x.x`, `192.168.x.x`, `169.254.x.x` (cloud metadata — AWS/GCP/Azure), `[::1]` — unless user explicitly tests localhost. Reject URLs with `@`. On rejection: report exact reason and stop, never silently skip.
- **No raw interpolation** — for the element-selector script, pass values via shell variables and `process.argv`, never via `node -e "..."` with user input.
- **Headless only** — never open a visible browser window.
- **Respect gated content** — don't screenshot login-walled or paywalled content the user doesn't own.
- **Default to cwd** — put output files in the current working directory unless the user specifies otherwise.
- **Close safely** — when using an inline script, wrap `browser.close()` in its own try/catch inside `finally` so a close-time error can't mask the real error.
