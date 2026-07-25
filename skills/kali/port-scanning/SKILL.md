---
name: port-scanning
description: Port scanning with naabu and nmap via kali-mcp. Use when asked to scan ports, discover services, find open ports, or map the attack surface of a target host.
---

# Port Scanning

> **Purpose**: Discover open TCP ports and running services on target hosts.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (create_asset, create_clue)
> **Input**: Target IP or hostname + project_id
> **Output**: Clues (open ports with service/version) + updated assets

## Boundaries
- **In scope**: TCP port discovery, service version detection, banner grabbing
- **Out of scope**: UDP scanning. Web application probing (hand off to web-probing skill). Exploitation (hand off to AC-Breach)

## Workflow
1. `exec` with `naabu -host <target> -top-ports 1000 -rate 1000`
2. Poll `get_job` until status "completed"
3. For each open port: `exec` with `nmap -sV -p <port> <target>` for service fingerprint
4. Record discoveries: `create_clue` with type="info_disclosure", content describing service, version, potential risks
5. Record new hosts: `create_asset` with IPs and discovered ports in description

## Error Recovery
| Failure | Action |
|---------|--------|
| naabu "no ports found" | Try `nmap -sT --top-ports 100 <target>` (full TCP connect) |
| nmap times out (>120s) | Reduce ports or add `--host-timeout 60s` |
| exec "failed" immediately | Check target reachability: `exec` with `ping -c 1 <target>` |

## Autonomy Rules
- **Proceed without asking**: 🟡 Scanning targets in scope. Standard port ranges. Recording findings.
- **Escalate to supervisor**: Target outside scope. Full 65535 scan needed. Unexpected service (potential backdoor).

## Verified Patterns
- `naabu -host 10.0.0.1 -top-ports 1000 -rate 1000` — broad scan, fast rate
- `nmap -sV -sC -p 22,80,443 target.com` — focused scan with scripts
- `nmap -sT -p- --min-rate 500 target` — full TCP connect for stealth
