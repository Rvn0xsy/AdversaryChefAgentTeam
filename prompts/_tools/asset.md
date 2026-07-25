## asset-mcp Tools (28 tools)

> **Server**: `http://{{MCP_ASSET_URL}}`

### Projects
| Tool | Description |
|------|-------------|
| `list_projects` | List all pentest projects |
| `get_project` | Get project by ID |
| `create_project` | Create a new project (name, description) |
| `update_project` | Update project info |
| `delete_project` | Delete a project |

### Assets
| Tool | Description |
|------|-------------|
| `list_assets` | List assets in a project by project_id |
| `search_assets` | Full-text search across name, IPs, domains, tech stack, description |
| `get_asset` | Get asset by ID |
| `create_asset` | Create asset with name, IPs, domains, tech_stack, scope, description |
| `update_asset` | Update asset info |
| `delete_asset` | Delete asset |

### Clues (Findings)
| Tool | Description |
|------|-------------|
| `list_clues` | List clues in a project by project_id |
| `search_clues` | Search clues by keyword + optional type/status filter |
| `get_clue` | Get clue by ID |
| `create_clue` | Create clue with type (vulnerability/info_disclosure/misconfig) and status (open/confirmed/false_positive/resolved) |
| `update_clue` | Update clue info (title, content, type, status) |
| `delete_clue` | Delete clue |

### Credentials
| Tool | Description |
|------|-------------|
| `list_credentials` | List credentials in a project by project_id |
| `get_credential` | Get credential by ID |
| `create_credential` | Create credential with credential_type, label, value, optional asset_id |
| `update_credential` | Update credential info |
| `delete_credential` | Delete credential |

### Work Logs
| Tool | Description |
|------|-------------|
| `list_work_logs` | List work logs in a project by project_id |
| `get_work_log` | Get work log by ID |
| `create_work_log` | Create work log entry (title, content) |
| `update_work_log` | Update work log |
| `delete_work_log` | Delete work log |

### Project Stats
| Tool | Description |
|------|-------------|
| `project_summary` | Rollup: asset count, clue count by type, credential count, worklog count |
