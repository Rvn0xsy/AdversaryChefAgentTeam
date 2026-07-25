# naabu — Fast Port Scanner

ProjectDiscovery's fast TCP port scanner. Uses half-open SYN scan by default.

## Common Flags

| Flag | Purpose |
|------|---------|
| `-host <ip>` | Single target IP |
| `-list targets.txt` | Scan from file |
| `-top-ports 100` | Top N ports (default: 100) |
| `-p 80,443,8080` | Specific ports |
| `-p -` | All 65535 ports |
| `-rate 1000` | Packets per second |
| `-silent` | Results only |

## Verified Patterns

```bash
naabu -host 10.0.0.1 -top-ports 1000 -rate 1000 -silent
naabu -host target.com -p 22,80,443,8080,8443 -silent
naabu -list targets.txt -top-ports 100 -rate 500 -silent
```

## Output Interpretation

```
10.0.0.1:22
10.0.0.1:80
10.0.0.1:443
```

Each line is `host:port`. Pipe to nmap for service detection.
