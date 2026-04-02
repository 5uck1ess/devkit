---
name: devkit:scrape
description: Scrape a URL and return clean Markdown (or Markdown-in-JSON). Uses Jina Reader, Firecrawl, or raw WebFetch.
---

# Web Scrape to Markdown

Fetch a URL and convert it to clean, LLM-ready Markdown. Supports multiple backends with automatic fallback.

## Usage

```
/devkit:scrape https://example.com
/devkit:scrape https://example.com --json
/devkit:scrape https://example.com https://other.com
/devkit:scrape https://example.com --backend firecrawl
```

## Arguments

- One or more URLs to scrape
- `--json` — wrap output as Markdown-in-JSON: `{ "url": "...", "title": "...", "markdown": "..." }`
- `--backend <name>` — force a specific backend (see below). Default: auto-detect.

## Backends (in priority order)

1. **Jina Reader** — prepend `https://r.jina.ai/` to the URL. Returns Markdown by default. Use if `JINA_API_KEY` is set (higher rate limits) or anonymously (~20 RPM).
2. **Firecrawl** — use if `FIRECRAWL_API_KEY` is set. Best for JS-heavy sites and anti-bot bypass.
3. **WebFetch fallback** — use Claude's built-in `WebFetch` tool. No API key needed, but returns raw content (less clean).

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

**Firecrawl (if FIRECRAWL_API_KEY is set and --backend firecrawl):**
```
Use Bash to call:
curl -s -X POST https://api.firecrawl.dev/v1/scrape \
  -H "Authorization: Bearer $FIRECRAWL_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"url":"{url}","formats":["markdown"]}'

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

- If Jina Reader returns an error or empty content, fall back to WebFetch
- If a URL is unreachable, report the error and continue with remaining URLs
- Never silently drop a URL — always report what happened

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

- Always try Jina Reader first — it's free and produces the cleanest output
- Respect rate limits — if scraping many URLs, add a brief pause between requests
- Don't scrape login-gated or paywall content — report it as inaccessible
- Strip cookie banners, nav bars, and footers when possible (Jina/Firecrawl do this automatically)
- If the user just wants to read a page, output Markdown. If they want to process it, use --json.
