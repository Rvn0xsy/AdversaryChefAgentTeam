## kali-mcp Tools (4 tools)

> **Server**: `http://{{MCP_KALI_URL}}`

| Tool | Description |
|------|-------------|
| `exec` | Run shell command asynchronously in Kali container. Returns job_id immediately. Supports nmap, sqlmap, gobuster, curl, netcat, dig, and any apt-installed tool. |
| `list_jobs` | List all jobs, optionally filter by status: running/completed/failed/killed/timed_out |
| `get_job` | Get job details: status + partial or full stdout/stderr. Partial output available while job is running. |
| `kill_job` | Force-kill a running job and its entire process group to clean up child processes |
