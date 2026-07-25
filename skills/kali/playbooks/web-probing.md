---
name: web-probing
description: HTTP probing with httpx. Use when asked to probe web servers, fingerprint HTTP services, check which URLs are alive, or detect web technologies.
---

# Web Probing

> **Purpose**: Detect live web servers and fingerprint their technology stack.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (update_asset, create_clue)
> **Input**: List of URLs/IPs:ports + project_id
> **Output**: Updated assets (tech_stack) + clues (server info, interesting headers)

## Boundaries
- **In scope**: HTTP/HTTPS probing, status code checks, server header capture, technology detection
- **Out of scope**: Directory brute-force (hand off to web-fuzzing skill). JS crawling (hand off to js-analysis skill). Vulnerability scanning (hand off to web-vuln-scan skill). Active exploitation (hand off to AC-Breach)

## Workflow
1. `exec` with `echo -e "<url1>\n<url2>" | httpx -status-code -server -title -tech-detect -silent`
2. Poll `get_job` until status "completed"
3. For each live URL: `update_asset` with discovered tech_stack
4. For interesting headers (Server, X-Powered-By, Set-Cookie): `create_clue` with type="info_disclosure"
5. For 401/403 responses: `create_clue` with type="info_disclosure" — these are target surfaces for AC-Breach

## Error Recovery
| Failure | Action |
|---------|--------|
| httpx returns 0 results | Verify URLs are reachable: `curl -I <url>` |
| All results are timeouts | Check if target blocks probing, try `-http-proxy http://proxy:port` |
| Technology detection fails | Fall back to `curl -I` and manually parse Server header |

## Autonomy Rules
- **Proceed without asking**: 🟡 Probing within scope. Recording technology stack. Identifying web surface.
- **Escalate to supervisor**: All URLs return 403/blocked (potential WAF). Discovery of exposed admin panels. Discovery of unauthenticated API docs (Swagger/GraphQL).

## Verified Patterns
- `cat urls.txt | httpx -status-code -server -title -tech-detect -silent -nc` — full probe, no color
- `httpx -u https://target.com -status-code -title -follow-redirects` — single URL with redirects
