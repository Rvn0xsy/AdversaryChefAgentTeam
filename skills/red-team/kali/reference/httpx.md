# httpx — HTTP Probe

ProjectDiscovery's HTTP probing tool. Tests URLs for aliveness, status codes, and technology detection.

## Common Flags

| Flag | Purpose |
|------|---------|
| `-u <url>` | Single URL |
| `-list targets.txt` | URLs from file |
| `-status-code` | Show HTTP status code |
| `-title` | Extract page title |
| `-server` | Show Server header |
| `-tech-detect` | Detect web technologies |
| `-follow-redirects` | Follow 30x |
| `-silent` | Results only |
| `-nc` | No color output |
| `-o results.txt` | Write to file |

## Verified Patterns

```bash
echo -e "https://target.com\nhttps://api.target.com" | httpx -status-code -server -title -tech-detect -silent
cat urls.txt | httpx -status-code -title -nc
httpx -u https://target.com -status-code -title -follow-redirects
```

## Output Interpretation

```
https://target.com [200] [nginx] [Dashboard]
https://api.target.com [401] [Apache] []
```

Format: URL | [status] | [Server header] | [title]. 200 = alive, 401/403 = exists but protected, timeout = no HTTP service.
