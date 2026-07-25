## kali-mcp Tools (5 tools)

> **Server**: `http://{{MCP_KALI_URL}}`

| Tool | Description |
|------|-------------|
| `exec` | Run shell command asynchronously in Kali container. Returns job_id immediately. Supports nmap, sqlmap, gobuster, curl, netcat, dig, and any apt-installed tool. |
| `job_wait` | **⭐ Preferred**: Wait for a job to complete (blocks until done or timeout). Use this instead of polling get_job — costs only 1 turn. |
| `get_job` | Get job details: status + partial or full stdout/stderr. Use for peeking at partial output, not for polling. |
| `list_jobs` | List all jobs, optionally filter by status: running/completed/failed/killed/timed_out |
| `kill_job` | Force-kill a running job and its entire process group to clean up child processes |
