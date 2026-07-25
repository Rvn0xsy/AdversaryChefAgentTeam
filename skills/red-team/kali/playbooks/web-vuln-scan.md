---
name: web-vuln-scan
description: Vulnerability scanning with nuclei. Use ONLY when explicitly ordered by Supervisor — never run automatically. IDS/IPS WILL detect this.
---

# Web Vulnerability Scanning

> **Purpose**: Run nuclei vulnerability templates against confirmed web targets.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (create_clue)
> **Input**: Target URL(s) + project_id + explicit Supervisor authorization
> **Output**: Clues (confirmed vulnerabilities with severity)

## Boundaries
- **In scope**: Running nuclei against explicitly authorized targets. Recording results as clues.
- **Out of scope**: Automatic scanning. Scanning without authorization. False positive triage (record all, let human triage).

## Workflow
1. Confirm explicit Supervisor authorization before running ANY nuclei command.
2. `exec` with `nuclei -u <url> -t /root/nuclei-templates/ -severity critical,high -silent`
3. Poll `get_job`. Expected runtime: 30-300s for a single URL.
4. Add `-severity medium` only if explicitly requested by Supervisor.
5. Record each finding: `create_clue` with type="vulnerability", content including template name, matched URL, severity.
6. Do NOT run `-severity low` or `-severity info` templates unless ordered — false positive mountain, not worth it.

## Error Recovery
| Failure | Action |
|---------|--------|
| nuclei "no templates found" | Verify `/root/nuclei-templates/` is available. Report to supervisor. |
| nuclei "target not reachable" | Double-check URL with `curl -I` |
| nuclei times out | Reduce template scope: `-t /root/nuclei-templates/cves/` |

## Autonomy Rules
- **Proceed without asking**: NEVER. Nuclei is 🔴 Intrusive. Every run requires explicit authorization.
- **Escalate to supervisor**: After scanning completes — report findings immediately. If target blocks the scan mid-way — report and stop.

## Verified Patterns
- `nuclei -u https://target.com -t /root/nuclei-templates/ -severity critical,high -silent` — standard scan
- `nuclei -list targets.txt -t /root/nuclei-templates/cves/ -severity critical -silent` — CVE-only batch scan
