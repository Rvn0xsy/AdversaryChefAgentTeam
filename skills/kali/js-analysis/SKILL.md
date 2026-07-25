---
name: js-analysis
description: JavaScript crawling and route extraction with katana. Use when asked to crawl JS, extract API endpoints, find hidden routes, or map a single-page application.
---

# JS Analysis

> **Purpose**: Crawl JavaScript-heavy websites to extract API endpoints and hidden routes.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (create_clue)
> **Input**: Target URL + project_id
> **Output**: Clues (extracted API routes, hidden endpoints, JS file URLs)

## Boundaries
- **In scope**: JS crawling, URL extraction, API route discovery, hidden endpoint identification
- **Out of scope**: Directory brute-force (hand off to web-fuzzing skill). Parameter fuzzing (hand off to web-fuzzing skill). Active vulnerability scanning — NEVER run nuclei from this skill. Authentication testing.

## Workflow
1. `exec` with `katana -u <target> -jc -kf all -silent` (crawl JS files, extract all URLs)
2. Poll `get_job` until status "completed"
3. Parse output — focus on:
   - API paths matching `/api/*`, `/v1/*`, `/graphql`
   - Route patterns containing `:id`, `{uuid}`
   - File upload endpoints
   - Authentication endpoints (`/login`, `/oauth`, `/token`)
4. Record extracted routes: `create_clue` with type="info_disclosure", content listing routes and source JS files
5. Flag interesting findings: unprotected admin routes, debug endpoints, exposed `.map` files

## Error Recovery
| Failure | Action |
|---------|--------|
| katana returns 0 results | Try without `-jc`: `katana -u <target> -silent` (crawl without JS parsing) |
| katana times out | Add `-m 50` to limit max URLs crawled |
| JS files blocked | Try `curl <target>/app.js` then manually search for route strings |

## Autonomy Rules
- **Proceed without asking**: 🟡 JS crawling within scope. Recording discovered routes. Flagging suspicious endpoints.
- **Escalate to supervisor**: Discovery of exposed debug/diagnostic endpoints. Discovery of hardcoded API keys or secrets in JS. Request to crawl outside stated scope.

## Verified Patterns
- `katana -u https://target.com -jc -kf all -silent -o routes.txt` — full JS crawl
- `katana -u https://target.com -d 3 -silent` — depth 3 crawl, no JS parsing
- `katana -u https://target.com -jc -em json,woff,png,svg -silent` — skip static assets
