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
- Non-`http(s)` schemes (`file://`, `ftp://`, `data:`)
- Private/reserved IPs: `localhost`, `127.0.0.1`, `0.0.0.0`, `10.x.x.x`, `172.16-31.x.x`, `192.168.x.x`, `169.254.x.x`, `[::1]` — unless the user explicitly wants to test localhost
- URLs containing `@` (credential-in-URL attack vector)

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

**Element selector** (Playwright CLI doesn't expose this directly, so use a small script):
```bash
node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto({url_as_json});
  await page.locator({selector_as_json}).screenshot({ path: {output_as_json} });
  await browser.close();
})();
"
```

Use `JSON.stringify`-safe values (never interpolate raw user strings into `-e`).

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
