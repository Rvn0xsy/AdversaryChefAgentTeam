# katana — Web Crawler

ProjectDiscovery's headless web crawler for URL extraction, JS parsing, and endpoint discovery.

## Common Flags

| Flag | Purpose |
|------|---------|
| `-u <url>` | Target URL |
| `-list urls.txt` | URLs from file |
| `-jc` | Parse JavaScript files for URLs |
| `-kf all` | Extract all URL kinds (not just navigational) |
| `-d 3` | Max crawl depth |
| `-m 100` | Max URLs to crawl |
| `-silent` | Results only |
| `-em json,woff,png` | Exclude file extensions |
| `-o routes.txt` | Write to file |
| `-json` | JSON output with source info |

## Verified Patterns

```bash
katana -u https://target.com -jc -kf all -silent
katana -u https://target.com -jc -kf all -silent -json
katana -u https://target.com -jc -em json,woff,png,svg -silent
katana -u https://target.com -d 3 -m 50 -silent
```

## Output Interpretation

```
https://target.com/api/v1/users
https://target.com/api/v1/login
https://target.com/assets/app.js
```

Focus on: `/api/*`, `/v1/*`, `/graphql` paths, route params (`:id`, `{uuid}`), auth endpoints (`/login`, `/oauth`, `/token`).
