---
name: screenshot
description: Capture a screenshot or PDF of a web page — use when asked to screenshot a URL, capture a page, take a picture of a website, generate a visual snapshot, or save a page as PDF. Supports full page, element selector, device emulation, and custom viewport.
---

# Web Page Screenshot

Capture screenshots and PDFs of web pages using Playwright.

## Step 1: Verify Playwright

Before doing anything else, check that Playwright is installed:

```bash
npx playwright --version
```

If the command fails or Playwright is not installed, stop and tell the user:

```
This requires Playwright (optional devkit dependency).

Install with:
  npx playwright install chromium

Chromium is ~170MB. You can also install firefox or webkit.
```

Do NOT attempt to work around a missing install. The skill cannot function without it.

## Step 2: Parse the Request

From the user's message, extract:

- **URL** — the page to capture (required; validate http/https only)
- **Output path** — where to save (default: `./screenshot-{timestamp}.png` in cwd)
- **Capture mode** — viewport (default), full-page, or specific element
- **Device emulation** — iPhone, Pixel, iPad, etc. (optional)
- **Viewport size** — custom width x height (optional)
- **Format** — PNG (default) or PDF
- **Selector** — CSS selector for element-only capture (optional)

## Step 3: Validate the URL

Reject:
- Non-`http(s)` schemes (`file://`, `ftp://`, `data:`, all others)
- Private/reserved IPs: `localhost`, `127.0.0.1`, `0.0.0.0`, `10.x.x.x`, `172.16-31.x.x`, `192.168.x.x`, `169.254.x.x` (cloud metadata — AWS/GCP/Azure), `[::1]` — unless the user explicitly wants to test localhost
- URLs containing `@` (credential-in-URL attack vector)

On rejection: **report the exact reason and stop**. Never silently skip an invalid URL — if the user passed a batch, name which URL failed validation and why.

## Step 4: Execute

Choose the command that matches the capture mode. Always quote the URL and output path.

**Viewport screenshot:**
```bash
npx playwright screenshot "{url}" "{output}"
```

**Full-page screenshot:**
```bash
npx playwright screenshot --full-page "{url}" "{output}"
```

**Custom viewport:**
```bash
npx playwright screenshot --viewport-size={width},{height} "{url}" "{output}"
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

## Step 5: Report

Output the absolute path and any metadata:

```
Screenshot saved: /abs/path/to/file.png
Size: 342 KB
Mode: full-page
```

If the page failed to load, report the error clearly with the HTTP status or timeout reason.

## Rules

- **URL validation first** — never launch the browser on an invalid or private URL
- **No shell injection** — never concatenate user URLs into shell strings without proper quoting
- **Headless only** — never open a visible browser window
- **Respect gated content** — don't screenshot login-walled or paywalled content the user doesn't own
- **Default to cwd** — put output files in the current working directory unless the user specifies otherwise
- **One browser, one close** — always `await browser.close()` to avoid orphaned processes
