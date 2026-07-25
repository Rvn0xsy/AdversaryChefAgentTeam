---
name: kali-toolkit
description: Kali Linux tool orchestration via kali-mcp. Routes reconnaissance tasks to the correct playbook and enforces tool escalation boundaries.
---

# Kali Toolkit

> **Purpose**: Match reconnaissance tasks to the correct tool playbook and enforce tool escalation boundaries.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (create_clue, create_asset, update_asset)
> **Input**: Task description + target + project_id
> **Output**: Executed playbook results recorded to asset-mcp

## Playbooks

When a task matches a playbook's trigger, follow its exact workflow. Do NOT improvise commands.

| Playbook | Trigger Keywords | Level |
|----------|-----------------|:-----:|
| `port-scanning` | "scan ports", "discover services", "find open ports", "what's running" | 🟡 |
| `web-probing` | "probe web", "fingerprint HTTP", "check alive", "detect tech" | 🟡 |
| `js-analysis` | "crawl JS", "extract routes", "find endpoints", "map API" | 🟡 |
| `web-fuzzing` | "fuzz dirs", "brute force", "find hidden", "discover params" | 🟡 |
| `web-vuln-scan` | "scan vulnerabilities", "run nuclei" | 🔴 |

All playbooks are in `playbooks/`. See `reference/<tool>.md` for command flags and usage examples.

## Tool Escalation Boundary

| Level | Tools | When |
|-------|-------|------|
| 🟢 Passive | curl -I, dig, ping | Always allowed |
| 🟡 Active | nmap -sV, gobuster, katana, ffuf, httpx | Within scope, record findings |
| 🔴 Intrusive | nuclei, sqlmap --os-shell, nmap scripts, password brute | ONLY on explicit Supervisor order |

You operate at 🟡 Active. NEVER upgrade to 🔴 without explicit authorization.

## Reference

When you need to verify command syntax or explore flags, consult `reference/<tool>.md`:

| Tool | Reference |
|------|-----------|
| naabu | `reference/naabu.md` |
| nmap | `reference/nmap.md` |
| httpx | `reference/httpx.md` |
| katana | `reference/katana.md` |
| gobuster | `reference/gobuster.md` |
| ffuf | `reference/ffuf.md` |
| nuclei | `reference/nuclei.md` |

## Tricks

See `tricks/` for technique-specific knowledge and tips (e.g., bypass techniques, WAF evasion, stealth scanning patterns).
