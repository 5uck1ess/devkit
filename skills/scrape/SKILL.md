---
name: scrape
description: Scrape a URL to clean Markdown — use when asked to scrape, fetch, extract content from, or read a webpage and convert it to Markdown. Uses Jina Reader, Playwright, Camoufox (TLS-spoofing Firefox), Scweet (X/Twitter), Firecrawl, or WebFetch.
---

# Web Scrape to Markdown

Fetch a URL and convert it to clean, LLM-ready Markdown. Supports multiple backends with automatic fallback.

## Usage

```
/devkit:scrape https://example.com
/devkit:scrape https://example.com --json
/devkit:scrape https://example.com https://other.com
/devkit:scrape https://example.com --backend playwright
```

## Arguments

- One or more URLs to scrape
- `--json` — wrap output as Markdown-in-JSON: `{ "url": "...", "title": "...", "markdown": "..." }`
- `--backend <name>` — force a specific backend (see below). Default: auto-detect.

## Backends (in priority order)

1. **Jina Reader** — prepend `https://r.jina.ai/` to the URL. Returns Markdown by default. Use if `JINA_API_KEY` is set (higher rate limits) or anonymously (~20 RPM). Best for articles and docs.
2. **Playwright** — use if `npx playwright --version` succeeds (optional dep, install with `npx playwright install chromium`). Best for JS-heavy SPAs and sites that block headless scrapers. Free and local — no API keys.
3. **Camoufox** — patched Firefox with leak-fixed JA3/TLS fingerprints. Use if `python -m camoufox version` succeeds (install: `pip install -U camoufox[geoip] && python -m camoufox fetch`). Free and local. Use when Playwright returns a challenge page (Cloudflare "Just a moment...", DataDome, PerimeterX). Auto-triggered when Playwright response body matches challenge signatures (`cf-chl-bypass`, `__cf_chl_`, `_Incapsula_Resource`).
4. **Scweet (X/Twitter only)** — specialized scraper for `x.com` / `twitter.com` URLs. Use if `SCWEET_COOKIES_PATH` is set (points at a JSON cookie file exported from a logged-in browser). Auto-routed when host is `x.com` or `twitter.com` — other backends don't work reliably on X post-API lockdown.
5. **Firecrawl** — use if `FIRECRAWL_API_KEY` is set. Paid API. Last-resort anti-bot bypass when stealth browsers still get blocked.
6. **WebFetch fallback** — use Claude's built-in `WebFetch` tool. No API key needed, but returns raw content (less clean).

## Execution

### Single URL

For each URL, try the highest-priority available backend:

**Jina Reader (default):**
```
Use WebFetch to fetch: https://r.jina.ai/{url}
Set header: Accept: text/markdown

If --json flag is set, instead fetch with:
  Accept: application/json
This returns: { "url": "...", "title": "...", "content": "..." }
where "content" is the Markdown.
```

**Playwright:**

Validate the URL first (see Rules: http/https only, no private IPs, no `@`).

Check availability: `npx playwright --version`. If not installed, tell the user once per session: `Playwright not installed. Install with: npx playwright install chromium`, then fall through to the next backend. The winning backend is always reported in Step 3 (Error Handling), so the user sees which one served the result.

Never build scripts via inline `-e "..."` strings. Write the script to a file that reads the URL from `process.argv[2]`, then invoke it:

```bash
DRIVER=$(mktemp -t devkit-scrape.XXXXXX.mjs)
trap 'rm -f "$DRIVER"' EXIT

cat > "$DRIVER" <<'EOF'
import { chromium } from 'playwright';

const url = process.argv[2];
if (!url) { console.error('usage: node devkit-scrape.mjs <url>'); process.exit(2); }

const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 });
  const title = await page.title();
  const html = await page.content();
  process.stdout.write(JSON.stringify({ title, html }));
} catch (err) {
  console.error('playwright failed:', err.message);
  process.exit(1);
} finally {
  try { await browser.close(); } catch (e) { console.error('playwright close error (browser may have crashed mid-scrape):', e.message); }
}
EOF

node "$DRIVER" "$URL"
```

Pass the URL as a shell variable (`URL=https://example.com`), never inline into the command string. The `.mjs` reads it from `process.argv[2]` so quotes, backticks, and `$()` in the URL can't escape.

Then convert HTML → Markdown. Prefer a local converter if available (pandoc, turndown). If none available, strip script/style/nav/footer tags and extract text from article/main/body.

**Camoufox (when Playwright returns a challenge page or `--backend camoufox`):**

Check availability: `python -m camoufox version`. If missing, print once per session: `Camoufox not installed. Install with: pip install -U camoufox[geoip] && python -m camoufox fetch`, then fall through.

Write the driver to a file (never inline `-c "..."` — same reasoning as Playwright). Pass the URL as an env var, never interpolated into the command string. Use `mktemp` rather than a hardcoded `/tmp/` path (portability + predictable-path attack surface):

```bash
DRIVER=$(mktemp -t devkit-camoufox.XXXXXX.py)
trap 'rm -f "$DRIVER"' EXIT

cat > "$DRIVER" <<'EOF'
import os, json, sys, traceback
from camoufox.sync_api import Camoufox

url = os.environ["DEVKIT_SCRAPE_URL"]
stage = "init"
try:
    with Camoufox(headless=True, humanize=True) as browser:
        page = browser.new_page()
        stage = "goto"
        page.goto(url, wait_until="domcontentloaded", timeout=30000)
        stage = "title"
        title = page.title()
        stage = "content"
        html = page.content()
        print(json.dumps({"title": title, "html": html}))
except Exception as err:
    print(json.dumps({"error": str(err), "stage": stage, "traceback": traceback.format_exc()[-2000:]}))
    sys.exit(1)
EOF

DEVKIT_SCRAPE_URL="$URL" python "$DRIVER"
```

Then convert HTML → Markdown (same converter as Playwright). If the JSON has an `error` field, treat the backend as failed and fall through.

**Challenge-page detection** (triggers auto-fallback Playwright → Camoufox): treat the Playwright result as blocked and retry with Camoufox when ANY of the following holds:

- Positive anti-bot signature present: `cf-chl-bypass`, `__cf_chl_`, `<title>Just a moment...</title>`, `_Incapsula_Resource`, `dd_cookie_test_`, `datadome`, `px-captcha`, `perimeterx` (case-insensitive match against the HTML)
- Body text under 200 chars AND no `<article>`/`<main>`/`<section>`/`<h1>` tag (skeleton/challenge shell, not a real small page)

Narrower than a size heuristic alone — valid small pages (404s, hand-rolled minimal HTML, API docs) keep their original backend instead of burning a Camoufox cycle.

**Scweet (X/Twitter — host is `x.com` or `twitter.com`):**

Check availability: `python -c "import Scweet"`. If missing: `pip install Scweet`. Requires `SCWEET_COOKIES_PATH` pointing at a JSON cookies file exported from a logged-in browser session (devkit convention: export with a cookie extension, store outside the repo, set `SCWEET_COOKIES_PATH=~/.config/devkit/x-cookies.json`). Do NOT hardcode auth tokens in env.

Scweet's Python API varies by version — the `scrape_tweets` / `scrape_profile` / `scrape_tweet` entry points differ across 2.x releases. Inspect the installed version with `python -c "import Scweet; print(Scweet.__version__)"` before wiring. Reference template (adapt to your installed Scweet version):

```bash
DRIVER=$(mktemp -t devkit-scweet.XXXXXX.py)
trap 'rm -f "$DRIVER"' EXIT

cat > "$DRIVER" <<'EOF'
import os, json, sys, traceback, re
from urllib.parse import urlparse

url = os.environ["DEVKIT_SCRAPE_URL"]
cookies_path = os.environ["SCWEET_COOKIES_PATH"]
stage = "init"
try:
    # Scweet 2.x: choose entry point by URL shape
    path = urlparse(url).path
    if re.match(r"^/[^/]+/status/\d+", path):
        stage = "import_tweet"
        from Scweet.user import get_tweet
        data = get_tweet(tweet_url=url, cookies_path=cookies_path, headless=True)
    elif re.match(r"^/[^/]+/?$", path):
        stage = "import_profile"
        from Scweet.user import get_user_information
        handle = path.strip("/")
        data = get_user_information(handles=[handle], cookies_path=cookies_path, headless=True)
    else:
        raise ValueError(f"unsupported x.com path: {path}")
    print(json.dumps({"data": data}, default=str))
except Exception as err:
    print(json.dumps({"error": str(err), "stage": stage, "traceback": traceback.format_exc()[-2000:]}))
    sys.exit(1)
EOF

DEVKIT_SCRAPE_URL="$URL" python "$DRIVER"
```

Output is structured tweet/profile data — render to Markdown (username + timestamp + text + engagement stats for tweets; bio + follower counts for profiles). If the JSON has an `error` field, fall back to Jina.

**Firecrawl (if FIRECRAWL_API_KEY is set and --backend firecrawl):**
```
Use Bash to call (use jq to safely construct JSON — never interpolate URLs directly):
jq -n --arg url "{url}" '{"url": $url, "formats": ["markdown"]}' | \
  curl -s -X POST https://api.firecrawl.dev/v1/scrape \
    -H "Authorization: Bearer $FIRECRAWL_API_KEY" \
    -H "Content-Type: application/json" \
    -d @-

Response contains: { "data": { "markdown": "...", "metadata": { "title": "..." } } }
```

**WebFetch fallback:**
```
Use WebFetch on the raw URL.
Note: output will be noisier than Jina/Firecrawl — no boilerplate removal.
```

### Multiple URLs

When given multiple URLs, scrape them in parallel:
- Launch one `researcher` agent per URL (max 5 concurrent)
- Each agent fetches its URL using the backend logic above
- Collect all results before outputting

### Error Handling

**Fallback chain (single linear order):** Jina Reader → Playwright → Camoufox → Firecrawl → WebFetch. Each backend is tried in order. Move to the next only on error, empty content, challenge-page detection, or missing dependency. **Exception:** URLs on `x.com` / `twitter.com` route directly to Scweet (skip the chain) when `SCWEET_COOKIES_PATH` is set; if Scweet is unavailable or fails, fall back to Jina.

- **Report which backend actually ran** for each URL, and (if fallbacks happened) why the earlier ones failed. Never silently substitute content — the user must know a paywall/cookie-wall/SPA-skeleton from one backend wasn't the real page fetched by another.
- If a URL is unreachable by every backend, report the error and continue with remaining URLs.
- Never silently drop or substitute a URL — always report what happened, with the winning backend named.

## Output

### Default (plain Markdown)

Output the extracted Markdown directly, prefixed with the source URL:

```
## Source: {url}

{extracted markdown content}
```

### With --json flag

Output structured JSON:

```json
{
  "url": "https://example.com",
  "title": "Page Title",
  "markdown": "# Heading\n\nContent here...",
  "backend": "jina",
  "timestamp": "2026-04-02T12:00:00Z"
}
```

For multiple URLs, output a JSON array.

## Rules

- **URL validation** — only accept `http://` and `https://` URLs. Reject `file://`, `ftp://`, `data:`, and all other schemes. Reject URLs targeting private/reserved IPs: `localhost`, `127.0.0.1`, `0.0.0.0`, `169.254.x.x` (cloud metadata — AWS/GCP/Azure), `10.x.x.x`, `172.16-31.x.x`, `192.168.x.x`, `[::1]`. Reject URLs containing `@` (credential-in-URL attacks). When rejecting, report the exact reason and stop — never silently skip a URL in a batch.
- **No shell injection** — never interpolate user URLs directly into shell command strings. Use `jq` to construct JSON payloads for curl. For Playwright, write scripts to files that read URLs from `process.argv`, and pass the URL as a shell variable — never inline into `-e` strings.
- **API keys from env only** — never hardcode API keys. Always reference `$JINA_API_KEY`, `$FIRECRAWL_API_KEY` from environment variables.
- Always try Jina Reader first — it's free and produces the cleanest output
- Respect rate limits — if scraping many URLs, add a brief pause between requests
- Don't scrape login-gated or paywall content — report it as inaccessible
- Strip cookie banners, nav bars, and footers when possible (Jina/Firecrawl do this automatically)
- If the user just wants to read a page, output Markdown. If they want to process it, use --json.
