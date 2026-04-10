---
name: scrape
description: Scrape a URL to clean Markdown — use when asked to scrape, fetch, extract content from, or read a webpage and convert it to Markdown. Uses Jina Reader, Playwright, Firecrawl, or WebFetch.
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
3. **Firecrawl** — use if `FIRECRAWL_API_KEY` is set. Paid API. Good for anti-bot bypass when Playwright isn't enough.
4. **WebFetch fallback** — use Claude's built-in `WebFetch` tool. No API key needed, but returns raw content (less clean).

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

**Playwright (if installed and --backend playwright, or as auto-fallback for JS-heavy sites):**

First validate the URL against the rules below (http/https only, no private IPs, no `@`).

Check availability: `npx playwright --version`. If not installed, tell the user once:
`Playwright not installed. Install with: npx playwright install chromium` — then fall through to the next backend AND record this in the backend-ran report so the user knows which backend actually served the result.

Never build scripts via inline `-e "..."` strings. Write the script to a file that reads the URL from `process.argv[2]`, then invoke it:

```bash
cat > /tmp/devkit-scrape.mjs <<'EOF'
import { chromium } from 'playwright';

const url = process.argv[2];
if (!url) { console.error('usage: node devkit-scrape.mjs <url>'); process.exit(2); }

const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 });
  const title = await page.title();
  const html = await page.content();
  process.stdout.write(JSON.stringify({ title, html }));
} catch (err) {
  console.error('playwright failed:', err.message);
  process.exit(1);
} finally {
  try { await browser.close(); } catch (e) { /* suppress close errors */ }
}
EOF

node /tmp/devkit-scrape.mjs "$URL"
```

Pass the URL as a shell variable (`URL=https://example.com`), never inline into the command string. The `.mjs` reads it from `process.argv[2]` so quotes, backticks, and `$()` in the URL can't escape.

Then convert HTML → Markdown. Prefer a local converter if available (pandoc, turndown). If none available, strip script/style/nav/footer tags and extract text from article/main/body.

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

**Fallback chain (single linear order):** Jina Reader → Playwright → Firecrawl → WebFetch. Each backend is tried in order. Move to the next only on error, empty content, or missing dependency.

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
