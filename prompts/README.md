# Agent Prompt Directory

Copy-paste ready agent instructions for Multica. All agents use `AC-` prefix.

## Quick Start

```bash
# 1. Expand tool placeholders
./scripts/expand-tools.sh prompts/echo-recon.md > /tmp/echo-final.md

# 2. Replace URL placeholders
sed -i 's|{{MCP_ASSET_URL}}|http://127.0.0.1:8081|g' /tmp/echo-final.md
sed -i 's|{{MCP_KALI_URL}}|http://127.0.0.1:8080|g' /tmp/echo-final.md

# 3. Create agent in Multica
multica agent create --name "AC-Echo" --instructions "$(cat /tmp/echo-final.md)" --runtime-id <codex-id>

# 4. Attach MCP config
multica agent update <agent-id> --mcp-config '{"mcpServers":{"asset":{"type":"http","url":"http://127.0.0.1:8081"},"kali":{"type":"http","url":"http://127.0.0.1:8080"}}}'
```

## Agent Roster

| Agent | File | MCP Required |
|-------|------|:---:|
| AC-Supervisor | `supervisor.md` | None |
| AC-Strategist | `strategist.md` | asset |
| AC-Echo | `echo-recon.md` | asset, kali |
| AC-Breach | `breach-exploit.md` | kali, asset |
| AC-Ghost | `ghost-mythic.md` | mythic, asset |
| AC-Path | `path-lateral.md` | mythic, kali, asset |
| AC-Forge | `forge-resource.md` | asset |
| AC-Quill | `quill-report.md` | asset |

## Placeholders

| Placeholder | Replace With |
|-------------|-------------|
| `{{WORKSPACE}}` | Multica workspace name |
| `{{MCP_ASSET_URL}}` | asset-mcp server address |
| `{{MCP_KALI_URL}}` | kali-mcp server address |
| `{{MCP_MYTHIC_URL}}` | mythic-mcp server address |
| `{{TOOLS_ASSET}}` | Auto-expanded by expand-tools.sh |
| `{{TOOLS_KALI}}` | Auto-expanded by expand-tools.sh |
| `{{TOOLS_MYTHIC}}` | Auto-expanded by expand-tools.sh |

## Adding a New MCP Server

1. Create `prompts/_tools/<name>.md` with tool table
2. Add `{{TOOLS_<NAME>}}` to relevant agent prompts
3. Run `./scripts/expand-tools.sh` — automatically picks up new tool file
4. Add test case to `prompts/_tests/`
