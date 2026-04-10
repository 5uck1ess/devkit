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

```bash
cat > /tmp/flow.mjs <<'EOF'
import { chromium } from 'playwright';

const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  await page.goto('{url}', { waitUntil: 'networkidle' });

  // Example: extract items from a JS-rendered list
  const data = await page.evaluate(() =>
    Array.from(document.querySelectorAll('.item')).map(el => ({
      title: el.querySelector('h3')?.textContent?.trim(),
      link: el.querySelector('a')?.href,
    }))
  );

  console.log(JSON.stringify(data, null, 2));
} finally {
  await browser.close();
}
EOF

node /tmp/flow.mjs
```

Never build scripts via long `-e "..."` strings with interpolated user input — write the script to a file, then run it.

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
cat > /tmp/flow.mjs <<'EOF'
import { chromium } from 'playwright';

const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  await page.goto('{login_url}');
  await page.fill('input[name="email"]', process.env.EMAIL);
  await page.fill('input[name="password"]', process.env.PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/dashboard');

  // Now do whatever the user wanted
  await page.goto('{target_url}');
  const content = await page.content();
  console.log(content);
} finally {
  await browser.close();
}
EOF

EMAIL='...' PASSWORD='...' node /tmp/flow.mjs
```

Never hardcode credentials. Always read from env vars or prompt the user.

## Step 4: Report

After running, report:

- What was done (pages visited, actions performed)
- What was extracted (or where it was saved)
- Any failures (timeouts, missing elements, auth errors)
- The script file path (if saved for reuse)

## Rules

- **URL validation** — only `http(s)`. Reject private IPs unless testing localhost is explicit. Reject URLs with `@`.
- **Credentials** — never hardcode. Read from env or ask the user. Never log them.
- **No raw interpolation** — write scripts to files, don't build via `-e "..."` with user input.
- **Always close the browser** — use `try/finally` to avoid orphaned processes.
- **Headless by default** — only use `headless: false` when user explicitly wants it (e.g., codegen, debugging).
- **Respect the site** — no CAPTCHA bypass, no hammering, no scraping the user doesn't own.
- **Auth state reuse** — for repeated runs against the same site, save `storageState` to avoid re-login.
- **Default timeout 30s** — if slow, use `page.waitForSelector()` rather than longer fixed waits, and explain why.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `"Executable doesn't exist"` | `npx playwright install chromium` |
| Timeout exceeded | Selector wrong or page slow — use `waitForSelector()` |
| Element not visible | `await locator.scrollIntoViewIfNeeded()` before action |
| Site blocks headless | Try user agent override; last resort: `headless: false` |
| Flaky data extraction | Use `networkidle` wait condition instead of `domcontentloaded` |
