# gobuster — Directory Brute-Forcer

Fast directory/file enumeration tool for web servers.

## Common Flags

| Flag | Purpose |
|------|---------|
| `dir` | Directory/file brute-force mode |
| `-u <url>` | Target URL |
| `-w <wordlist>` | Path to wordlist |
| `-t 20` | Thread count |
| `-q` | Quiet mode (no banner) |
| `-x php,html,txt` | File extensions to try |
| `--delay 500ms` | Delay between requests |
| `-o results.txt` | Write output to file |

## Verified Patterns

```bash
gobuster dir -u https://target.com -w /data/dictionaries/dir/common.txt -t 20 -q
gobuster dir -u https://target.com -w /data/dictionaries/api/common.txt -t 20 -q
gobuster dir -u https://target.com -w /data/dictionaries/dir/common.txt -t 5 --delay 500ms -q
```

## Output Interpretation

```
/admin (Status: 403)
/login (Status: 200)
/.git (Status: 403)
```

Status codes: 200 = accessible, 403 = exists but forbidden, 301/302 = redirect. 404 = not found. All 404 = site may use routing (try ffuf instead).
