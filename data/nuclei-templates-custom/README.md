# Custom Nuclei Templates

Place custom YAML nuclei templates here. These will be merged over the official templates
when running `scripts/update-nuclei-templates.sh`.

## Directory Structure

Match the official nuclei-templates structure:

```
cves/           — CVE-specific templates
vulnerabilities/ — General vulnerability templates
exposures/      — Exposure/misconfiguration templates
misconfig/      — Misconfiguration templates
workflows/      — Multi-step workflow templates
```

## Template Format

Standard nuclei YAML format:

```yaml
id: my-custom-check
info:
  name: My Custom Vulnerability Check
  author: rvn0xsy
  severity: medium
  description: Detects custom misconfiguration
requests:
  - method: GET
    path:
      - "{{BaseURL}}/internal/status"
    matchers:
      - type: word
        words:
          - "internal service"
```

## Updating Custom Templates

1. Add/modify `.yaml` files in this directory
2. Run `./scripts/update-nuclei-templates.sh` to merge into official templates
3. Rebuild kali-mcp Docker image
