## mythic-mcp Tools (14 tools)

> **Server**: `http://{{MCP_MYTHIC_URL}}`

### Callbacks
| Tool | Description |
|------|-------------|
| `mythic_list_callbacks` | List all active agent callbacks with host, user, IPs, PID |
| `mythic_get_callback` | Get callback details by display_id |
| `mythic_get_callback_commands` | List all loaded commands + parameter groups for a callback. ALWAYS call this before tasking an unfamiliar callback. |

### Tasking
| Tool | Description |
|------|-------------|
| `mythic_issue_task` | Issue a command to a callback. Uses exact command name and parameters from get_callback_commands. Returns task_id. |
| `mythic_list_tasks` | List all tasks for a callback, most recent first |
| `mythic_get_task_status` | Check task status without blocking |
| `mythic_wait_for_task` | Block until task completes then return output. Use for fast foreground commands only. |
| `mythic_get_task_output` | Get full decoded output of a completed task |

### Files
| Tool | Description |
|------|-------------|
| `mythic_list_files` | List files in Mythic file store |
| `mythic_upload_file` | Upload a local file to Mythic |
| `mythic_download_file` | Download a file from Mythic by agent_file_id |
| `mythic_delete_file` | Delete a file from Mythic |

### Payloads
| Tool | Description |
|------|-------------|
| `mythic_create_payload` | Create a new payload/agent (exe, dll, shellcode, etc.) |
| `mythic_get_payload` | Get payload details by UUID |
