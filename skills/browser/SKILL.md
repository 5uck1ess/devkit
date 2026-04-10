---
name: browser
description: Automate a web browser — use when asked to click buttons, fill forms, navigate multi-step flows, extract data from JS-rendered pages, log into a site, record user interactions, or test a web app in a real browser. Use for anything beyond simple article scraping.
---

# Browser Automation

Drive a real browser via Playwright to interact with pages — click, fill forms, extract data from SPAs, run multi-step flows, or record user interactions.

## When to Use This vs Related Skills

| User asks for... | Use |
|---|---|
| "fetch this article as markdown" | `scrape` (Jina is faster for static content) |
| "screenshot this page" | `screenshot` |
| "extract data from this JS-heavy page" | **`browser`** |
| "log into X and download Y" | **`browser`** |
| "fill this form and submit" | **`browser`** |
| "record me clicking through this flow" | **`browser`** (codegen) |
| "test my web app end-to-end" | **`browser`** |

## Step 1: Verify Playwright

```bash
npx playwright --version
```

If not installed, stop and tell the user:

```
This requires Playwright (optional devkit dependency).

Install with:
  npx playwright install chromium
```

Do not attempt workarounds.

## Step 2: Parse the Request

Understand what the user wants:

- **Target URL(s)** — starting page
- **Actions** — navigate, click, fill, extract, screenshot, wait
- **Data to extract** — what fields, what format (JSON usually)
- **Auth** — are credentials needed? (prompt if user hasn't provided)
- **Repeatability** — one-off or should this become a reusable script?

If the user's request is vague ("scrape this site"), clarify:
- Which data fields do you want?
- Does it require login?
- Is it a one-shot or will you re-run this?

## Step 3: Choose the Right Mode

### Mode A — Codegen (recording)

When the user wants to figure out selectors or hand off a repeatable flow:

```bash
npx playwright codegen {url}
```

Opens a browser. User interacts. Playwright prints the equivalent script. Best for:
- Complex pages where selectors aren't obvious
- Building reusable flows
- Teaching users how Playwright works

### Mode B — Inline script (one-off)

For quick, throwaway extractions, write a small script to `/tmp/` and run it:

Write the script to a file that reads the URL from `process.argv`, then pass it via a shell variable. Never interpolate user input into the script source itself.

```bash
cat > /tmp/devkit-browser-flow.mjs <<'EOF'
import { chromium } from 'playwright';

const url = process.argv[2];
if (!url) { console.error('usage: node devkit-browser-flow.mjs <url>'); process.exit(2); }

const browser = await chromium.launch();
let exitCode = 0;
try {
  const page = await browser.newPage();
  await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 });

  // Example: extract items from a JS-rendered list
  const data = await page.evaluate(() =>
    Array.from(document.querySelectorAll('.item')).map(el => ({
      title: el.querySelector('h3')?.textContent?.trim(),
      link: el.querySelector('a')?.href,
    }))
  );

  console.log(JSON.stringify(data, null, 2));
} catch (err) {
  console.error('flow failed:', err.message);
  exitCode = 1;
} finally {
  try { await browser.close(); } catch (e) { /* suppress close errors so the real error propagates */ }
}
process.exit(exitCode);
EOF

URL=https://example.com node /tmp/devkit-browser-flow.mjs "$URL"
```

Never build scripts via long `-e "..."` strings with interpolated user input — write the script to a file that reads args from `process.argv`, then pass values as shell variables.

### Mode C — Persistent test (for web apps)

If the user is building a web app and wants E2E tests:

```bash
# Scaffold Playwright test framework
npm init playwright@latest

# Generates playwright.config.ts and tests/ directory
# Run with:
npx playwright test
```

Then write test specs in `tests/*.spec.ts`.

### Mode D — Form fill + auth flow

For "log in and grab something":

```bash
cat > /tmp/devkit-browser-auth.mjs <<'EOF'
import { chromium } from 'playwright';

const loginUrl = process.argv[2];
const targetUrl = process.argv[3];
if (!loginUrl || !targetUrl) {
  console.error('usage: node devkit-browser-auth.mjs <login_url> <target_url>');
  process.exit(2);
}

const browser = await chromium.launch();
let exitCode = 0;
try {
  const page = await browser.newPage();
  await page.goto(loginUrl);
  await page.fill('input[name="email"]', process.env.EMAIL);
  await page.fill('input[name="password"]', process.env.PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/dashboard');

  await page.goto(targetUrl);
  const content = await page.content();
  console.log(content);
} catch (err) {
  console.error('auth flow failed:', err.message);
  exitCode = 1;
} finally {
  try { await browser.close(); } catch (e) { /* suppress close errors */ }
}
process.exit(exitCode);
EOF

EMAIL="$EMAIL" PASSWORD="$PASSWORD" node /tmp/devkit-browser-auth.mjs "$LOGIN_URL" "$TARGET_URL"
```

Never hardcode credentials. Always read from env vars or prompt the user. Pass URLs as shell variables, never inline into the script source.

## Step 4: Report

After running, report:

- What was done (pages visited, actions performed)
- What was extracted (or where it was saved)
- Any failures (timeouts, missing elements, auth errors)
- The script file path (if saved for reuse)

## Rules

- **URL validation** — only accept `http://` and `https://`. Reject `file://`, `ftp://`, `data:`, and all other schemes. Reject private/reserved IPs: `localhost`, `127.0.0.1`, `0.0.0.0`, `169.254.x.x` (cloud metadata — AWS/GCP/Azure), `10.x.x.x`, `172.16-31.x.x`, `192.168.x.x`, `[::1]` — unless the user is explicitly testing localhost. Reject URLs containing `@` (credential-in-URL attacks). On rejection: **report the exact reason and stop** — never silently skip.
- **Credentials** — never hardcode. Read from env or ask the user. Never log them.
- **No raw interpolation** — write scripts to files that read args from `process.argv`, then pass values via shell variables. Never build via `-e "..."` with interpolated user input.
- **Always close the browser safely** — use `try/catch/finally` with the `close()` call in its own try/catch, so a close-time error never masks the real error.
- **Headless by default** — only use `headless: false` when user explicitly wants it (e.g., codegen, debugging).
- **Respect the site** — no CAPTCHA bypass, no hammering, no scraping the user doesn't own.
- **Auth state reuse** — for repeated runs against the same site, save `storageState` to avoid re-login.
- **Default timeout 30s** — if slow, use `page.waitForSelector()` rather than longer fixed waits, and explain why.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `"Executable doesn't exist"` | `npx playwright install chromium` |
| Timeout exceeded | **Report the timeout to the user first with the URL and selector** — do not retry silently. Then investigate: wrong selector, page slow, content gated by JS that never fires. Use `waitForSelector()` with a sensible timeout rather than longer fixed waits. |
| Element not visible | `await locator.scrollIntoViewIfNeeded()` before action |
| Site blocks headless | Try user agent override; last resort: `headless: false` |
| Flaky data extraction | Use `networkidle` wait condition instead of `domcontentloaded` |
