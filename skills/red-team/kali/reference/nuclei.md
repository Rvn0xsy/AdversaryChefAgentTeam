# nuclei — Vulnerability Scanner

ProjectDiscovery's template-based vulnerability scanner. 🔴 Intrusive — requires explicit supervisor authorization.

## Common Flags

| Flag | Purpose |
|------|---------|
| `-u <url>` | Single target URL |
| `-list targets.txt` | Targets from file |
| `-t <path>` | Template or template directory |
| `-severity critical,high` | Severity filter |
| `-tags cve` | Template tag filter |
| `-silent` | Results only |
| `-o results.txt` | Write to file |

## Verified Patterns

```bash
nuclei -u https://target.com -t /root/nuclei-templates/ -severity critical,high -silent
nuclei -list targets.txt -t /root/nuclei-templates/cves/ -severity critical -silent
nuclei -u https://target.com -t /root/nuclei-templates/ -severity critical,high,medium -silent
```

## Output Interpretation

```
[critical] CVE-2021-44228 [http] https://target.com/api/log
[high] exposed-panel [http] https://target.com/phpmyadmin
```

Format: [severity] | [template-name] | [protocol] | [matched-URL]. Record all findings — false positive triage is a human task.
