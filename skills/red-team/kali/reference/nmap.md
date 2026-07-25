# nmap — Network Mapper

Standard network discovery and security auditing tool.

## Common Flags

| Flag | Purpose |
|------|---------|
| `-sV` | Service/version detection |
| `-sC` | Default NSE scripts |
| `-sT` | Full TCP connect (no raw sockets) |
| `-p 22,80,443` | Specific ports |
| `-p-` | All ports |
| `--top-ports 100` | Top N ports |
| `--host-timeout 60s` | Per-host timeout |
| `--min-rate 500` | Minimum packets/sec |
| `-oN output.txt` | Normal output to file |

## Verified Patterns

```bash
nmap -sV -p 22,80,443 target.com
nmap -sT -p- --min-rate 500 10.0.0.1
nmap -sV -sC -p 80,443 --script http-enum target.com
nmap -sn 10.0.0.0/24
```

## Output Interpretation

```
PORT    STATE SERVICE  VERSION
22/tcp  open  ssh      OpenSSH 8.4p1
80/tcp  open  http     nginx 1.18.0
```

Format: PORT | STATE | SERVICE | VERSION. `open` = accessible, `filtered` = firewall/blocked, `closed` = nothing listening.
