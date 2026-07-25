# acactl CLI Design

## Overview

`acactl` is the unified CLI for AdversaryChefAgentTeam. It manages infrastructure
lifecycle (nexus-mcp, kali-mcp, acasched) and provides task execution and
observability.

## 1. Command Structure

```
acactl
├── up         Start all infrastructure (nexus + kali + acasched)
├── down       Stop all services
├── status     Check service health
├── run        Create penetration task + real-time observation
├── tasks      List tasks for a project
├── logs       Replay Agent execution (formatted or raw stream-json)
└── projects   Manage projects (create/list)
```

## 2. Process Lifecycle

### Data Directory

```
~/.aca/
├── bin/
│   ├── nexus-mcp
│   └── acasched
├── data/
│   ├── nexus.db
│   └── acasched.db
├── logs/
│   ├── nexus-mcp.log
│   ├── acasched.log
│   └── tasks/
│       └── <task_id>.jsonl
├── pids/
│   ├── nexus-mcp.pid
│   └── acasched.pid
└── config.yaml
```

### up

```
acactl up [--force-rebuild]

① Pre-check
  - Check ports 8080, 8081, 9090 free (or owned by existing aca processes)
  - Check podman running
  - Build go binaries if missing or --force-rebuild: go build → ~/.aca/bin/
  - Check kali-mcp image exists in podman

② Start (sequentially, each waits for health check)
  - nexus-mcp: spawn ~/.aca/bin/nexus-mcp -db ~/.aca/data/nexus.db
    → poll :8081/health (max 5s)
  - kali-mcp: podman run -d --name kali-mcp --cap-add NET_RAW ...
    → poll :8080/health (max 10s)
  - acasched: spawn ~/.aca/bin/acasched -db ~/.aca/data/acasched.db
    -nexus-mcp http://127.0.0.1:8081 -kali-mcp http://127.0.0.1:8080
    -prompts <project>/prompts/ -log-dir ~/.aca/logs/tasks/
    → poll :9090/health (max 5s)

③ Write pid files

④ Output status table:
  ┌────────────┬─────────┬───────┐
  │ SERVICE    │ STATUS  │ PORT  │
  ├────────────┼─────────┼───────┤
  │ nexus-mcp  │ running │ 8081  │
  │ kali-mcp   │ running │ 8080  │
  │ acasched   │ running │ 9090  │
  └────────────┴─────────┴───────┘
```

### down

```
acactl down

① Stop in reverse order:
  - acasched: read pid → kill -TERM → wait 3s → kill -9 if still alive → remove pid
  - kali-mcp: podman stop kali-mcp && podman rm kali-mcp
  - nexus-mcp: read pid → kill -TERM → wait 3s → kill -9 if still alive → remove pid

② Verify all ports released
③ Output: All services stopped
```

### status

```
acactl status

① GET :8081/health + :8080/health + :9090/health
② Output table with status indicators (● running / ○ stopped / ✖ unhealthy)
```

## 3. Task Operations

### run

```
acactl run --goal "penetration test target.com" [--project proj_xxx] [--scope "*.target.com"]

① If no --project: POST /api/projects {name: goal摘要}
② POST /api/tasks {project_id, agent:"supervisor", title:goal, description:goal, created_by:"cli"}
③ Print task_id + "Waiting for Supervisor..."
④ Poll GET /api/tasks/:id every 2s until status is terminal (done/failed/timeout)
⑤ Each poll: print new sub-tasks and their summaries as they complete
⑥ Final output: execution summary with child task results
```

### tasks

```
acactl tasks [--project proj_xxx] [--status pending|running|done|failed]

① GET /api/tasks?project_id=X&status=Y
② Table output:
  TASK ID            AGENT       STATUS    TITLE                    DURATION
  task_xxx           echo        done      Recon scanme.nmap.org    42s
  task_yyy           breach      running   Exploit SQLi at /login   ...
```

### projects

```
acactl projects create --name "靶场测试" [--description "..."]
acactl projects list
```

## 4. Logs

### Data Source

acasched goose runner writes stream-json to `--log-dir/<task_id>.jsonl`:
- `cmd.Stdout` tee'd to file
- One JSON object per line (goose `--output-format stream-json`)

acasched new endpoint:
```
GET /api/tasks/:id/logs         → return file contents
GET /api/tasks/:id/logs?follow  → SSE streaming
```

### logs command

```
acactl logs <task-id> [--follow] [--raw]

① GET /api/tasks/:id/logs (or SSE if --follow)
② Parse stream-json lines
③ Format: see §5 Formatting Rules
④ If --raw: output raw stream-json unchanged
```

### Formatting Rules

| stream-json type | Display |
|---|---|
| `tool_call` | `▸ [mcp-server] tool_name` + key params + indent |
| | Highlight: command for exec, url for endpoint_create, port+name for service_create |
| `tool_result` | Indented summary: ≤ 200 lines raw, >200 lines → `[truncated, -N lines]` |
| | For exec results: show stripped tool output |
| `assistant` | `✦ AgentName` + reasoning text |
| Tool timing | `⏱ X.Xs` from timestamp diff |

### --follow mode

- SSE connect to acasched → real-time new lines
- Auto-exit when task status is terminal
- Ctrl-C disconnects without affecting task

## 5. Configuration

### config.yaml (optional, defaults embedded)

```yaml
ports:
  nexus: 8081
  kali: 8080
  acasched: 9090

data_dir: ~/.aca
project_root: .  # where go.mod/go.work lives

kali:
  image: kali-mcp
  cap_add: [NET_RAW, NET_ADMIN]
```

### CLI flags override config

```
acactl up --nexus-port 18081 --data-dir /opt/aca
```

## 6. Module

```
cmd/acactl/
├── main.go              # Entry point
├── commands/
│   ├── up.go
│   ├── down.go
│   ├── status.go
│   ├── run.go
│   ├── tasks.go
│   ├── logs.go
│   └── projects.go
├── lifecycle/
│   ├── manager.go       # ProcessManager: Start, Stop, HealthCheck
│   └── ports.go         # Port conflict detection
└── display/
    ├── table.go         # Status/output tables
    └── stream.go        # stream-json formatter
```

Module: `adversarychef/acactl`

## 7. acasched Changes

### New flag

```
-log-dir string   Task log directory (default: ~/.aca/logs/tasks/)
```

### goose runner: stream-json persistence

```go
// Execute() modified: tee stdout to log file
logPath := filepath.Join(r.LogDir, task.ID + ".jsonl")
logFile, _ := os.Create(logPath)
cmd.Stdout = logFile
cmd.Stderr = logFile
err := cmd.Run()
// Post-parse from logFile
```

### New API endpoint

```
GET /api/tasks/:id/logs
  - Read {LogDir}/{taskID}.jsonl, return as JSON array or multipart
  - ?follow=true: SSE stream, tail new lines, send {"event":"done"} when task terminal
```

Route registration in `server.go`:
```go
mux.HandleFunc("GET /api/tasks/{id}/logs", handleTaskLogs(s, logDir))
```
