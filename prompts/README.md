# Agent Prompt Directory

Copy-paste ready agent instructions for Multica. All agents use `AC-` prefix.

## Quick Start

```bash
# 1. Expand tool placeholders (from prompts/red-team/ directory)
./scripts/expand-tools.sh prompts/red-team/echo-recon.md > /tmp/echo-final.md

# 2. Replace URL placeholders
sed -i 's|{{MCP_ASSET_URL}}|http://127.0.0.1:8081|g' /tmp/echo-final.md
sed -i 's|{{MCP_KALI_URL}}|http://127.0.0.1:8080|g' /tmp/echo-final.md

# 3. Create agent in Multica
multica agent create --name "AC-Echo" --instructions "$(cat /tmp/echo-final.md)" --runtime-id <codex-id>

# 4. Attach MCP config
multica agent update <agent-id> --mcp-config '{"mcpServers":{"asset":{"type":"http","url":"http://127.0.0.1:8081"},"kali":{"type":"http","url":"http://127.0.0.1:8080"}}}'
```

Each agent's prompt file includes a `Skills` header field that declares the skill directory (under `skills/<squad>/`) from which the agent loads its operational knowledge — playbooks, reference data, and tricks.

## Agent Roster

| Agent | File | Skills | MCP Required |
|-------|------|--------|:---:|
| AC-Supervisor | `red-team/supervisor.md` | `red-team/_none` | nexus |
| AC-Strategist | `red-team/strategist.md` | `red-team/_none` | asset |
| AC-Echo | `red-team/echo-recon.md` | `red-team/kali` | asset, kali |
| AC-Breach | `red-team/breach-exploit.md` | `red-team/kali` | kali, asset |
| AC-Ghost | `red-team/ghost-mythic.md` | — | mythic, asset |
| AC-Path | `red-team/path-lateral.md` | `red-team/kali` | mythic, kali, asset |
| AC-Forge | `red-team/forge-resource.md` | — | asset |
| AC-Quill | `red-team/quill-report.md` | — | asset |

## Configuration Files

### `_mcp-registry.yaml`
Maps MCP names to URLs. Agents declare which MCPs they need via the `Requires` field in their prompt header.
Supports both local (`http://127.0.0.1`) and internet (`https://api.example.com`) URLs.

```yaml
nexus-mcp:  "http://127.0.0.1:8081"
kali-mcp:   "http://127.0.0.1:8080"
mythic-mcp: "http://127.0.0.1:8082"
```

### `_squads.yaml`
Declares available squads and their directory mappings. The runner reads this to discover agents, skills, and coordination logic.

```yaml
squads:
  red-team:
    description: "攻防厨师团 — Red Team Operations"
    prompt_dir: "red-team"
    skill_dir: "red-team"
```

### Adding a New Squad

To add a new squad to the system, follow this zero-code process:

1. **Create** `prompts/<squad>/` with agent `.md` files
2. **Create** `skills/<squad>/` with squad-specific skills (playbooks, reference, tricks)
3. **Add entry** to `_squads.yaml` with `prompt_dir` and `skill_dir` mappings
4. **Add MCP entries** to `_mcp-registry.yaml` for any new MCP servers the squad requires
5. **Done** — no code changes needed

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
