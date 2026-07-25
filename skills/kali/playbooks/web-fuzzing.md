---
name: web-fuzzing
description: Directory brute-force and parameter fuzzing with gobuster and ffuf. Use when asked to find hidden directories, brute-force API paths, fuzz parameters, or discover hidden content.
---

# Web Fuzzing

> **Purpose**: Discover hidden directories and test API parameters on web targets.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (create_clue)
> **Input**: Target URL + project_id
> **Output**: Clues (discovered paths, 200 responses with interesting content)

## Boundaries
- **In scope**: Directory brute-force, API path discovery, parameter fuzzing, file extension testing
- **Out of scope**: Password brute-force (🟡 only on explicit order — use pass/top100.txt). Vulnerability scanning — NEVER run nuclei. Exploitation (hand off to AC-Breach)

## Dictionaries
All dictionaries are mounted at `/data/dictionaries/`:
| Path | Content | When |
|------|---------|------|
| `/data/dictionaries/dir/common.txt` | Common web directories | Directory brute-force |
| `/data/dictionaries/api/common.txt` | Common API paths | API endpoint discovery |
| `/data/dictionaries/param/common.txt` | Common parameter names | Parameter fuzzing |
| `/data/dictionaries/pass/top100.txt` | Top 50 weak passwords | Password brute (🟡 only on order) |

## Workflow

### Directory Discovery
1. `exec` with `gobuster dir -u <url> -w /data/dictionaries/dir/common.txt -t 20 -q`
2. Poll `get_job`. Expected runtime: 60-180s depending on rate limiting.
3. Record 200/403 responses: `create_clue` with type="info_disclosure". 403 means the path exists but is protected.
4. Record 301/302 redirects: the redirect target may reveal internal architecture.

### Parameter Fuzzing
1. `exec` with `ffuf -u <url>?FUZZ=1 -w /data/dictionaries/param/common.txt -mc 200,403,500 -t 20`
2. Poll `get_job`
3. Any difference in response (200/403/500 vs baseline) indicates the parameter exists
4. Record discovered parameters: `create_clue`, include the parameter name and response difference

### API Path Discovery
1. `exec` with `gobuster dir -u <url> -w /data/dictionaries/api/common.txt -t 20 -q`
2. Same analysis as directory discovery

## Error Recovery
| Failure | Action |
|---------|--------|
| gobuster all 404 | Target may use routing (all paths return 200). Try `ffuf -u <url>/FUZZ -w dict -fc 404` |
| Rate limited (429) | Reduce `-t 5` and retry with delay `--delay 500ms` |
| ffuf baseline unknown | Run `ffuf -u <url>?FUZZ=1 -w dict -mc all` first to see default response pattern |

## Autonomy Rules
- **Proceed without asking**: 🟡 Directory/API fuzzing within scope. Parameter discovery. Recording findings.
- **Escalate to supervisor**: Discovery of `/admin` or unprotected management panels. Discovery of `.git/`, `.env`, or `backup.zip`. Password brute-force requests. Rate limiting from target (they're detecting us).

## Verified Patterns
- `gobuster dir -u https://target.com -w /data/dictionaries/dir/common.txt -t 20 -q` — standard brute
- `ffuf -u https://target.com/api/user?id=FUZZ -w /data/dictionaries/param/common.txt -mc 200 -t 20` — param fuzz with IDOR potential
- `ffuf -u https://target.com/FUZZ -w /data/dictionaries/dir/common.txt -mc 200,403 -t 20` — alternative to gobuster
