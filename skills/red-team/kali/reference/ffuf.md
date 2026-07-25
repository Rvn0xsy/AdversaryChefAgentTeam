# ffuf — Fuzz Faster U Fool

Fast web fuzzer for parameter discovery, path brute-forcing, and content discovery.

## Common Flags

| Flag | Purpose |
|------|---------|
| `-u <url>` | Target URL with FUZZ keyword |
| `-w <wordlist>` | Path to wordlist |
| `-mc 200,403` | Match status codes |
| `-fc 404` | Filter status codes |
| `-t 20` | Thread count |
| `-maxtime 300` | Max runtime in seconds |

## Verified Patterns

```bash
ffuf -u https://target.com/FUZZ -w /data/dictionaries/dir/common.txt -mc 200,403 -t 20
ffuf -u https://target.com/api/user?id=FUZZ -w /data/dictionaries/param/common.txt -mc 200 -t 20
ffuf -u https://target.com/FUZZ -w /data/dictionaries/api/common.txt -mc 200 -t 20
```

## Output Interpretation

```
FUZZ                    Status   Size
admin                   403      162
api/v1                  200      845
```

First column = matched word. Use `-mc all` first to see baseline response sizes, then filter with `-mc` or `-fc`.
