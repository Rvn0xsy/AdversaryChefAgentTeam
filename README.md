# AdversaryChef Agent Team

Event-driven, parallel red-team agent squad. Agents collaborate via nexus-mcp graph, orchestrated by an acasched scheduler — no human in the loop.

## Architecture

```
                            ┌──────────────┐
                            │   acactl     │  one-shot CLI
                            │  run -goal   │  builds + starts all services
                            └──────┬───────┘
                                   │
                    ┌──────────────┴──────────────┐
                    │         acasched            │
                    │   :9090  event loop         │
                    │   dispatcher · fallback poll │
                    └──────┬──────────┬───────────┘
                           │          │
              ┌────────────▼──┐  ┌────▼───────────┐
              │   nexus-mcp   │  │    kali-mcp     │
              │    :8081      │  │    :8080        │
              │  graph DB     │  │  async jobs     │
              │  webhooks →   │  │  nmap/curl/ssh  │
              └───────┬───────┘  └────────────────┘
                      │
         ┌────────────┼────────────┐
         │            │            │
    ┌────▼───┐  ┌─────▼────┐  ┌───▼──────┐
    │ AC-Echo│  │AC-Breach │  │AC-Forge  │ ... goose containers
    │ recon  │  │ exploit  │  │ infra    │     docker run --rm
    └────────┘  └──────────┘  └──────────┘
```

**Event flow**: Agent writes to nexus-graph → nexus POSTs webhook to acasched → Supervisor wakes up → evaluates state → dispatches next agents in parallel.

## Red Team Squad

| Agent | Role | Trigger |
|-------|------|---------|
| **AC-Supervisor** | Read-only coordinator. Evaluates nexus graph, dispatches agents | Graph change |
| **AC-Echo** | Recon: port scan, DNS, HTTP probe, JS crawl | hosts == 0 |
| **AC-Breach** | Exploit verification, PoC, initial access | host + service + evidence |
| **AC-Ghost** | C2 operations via Mythic | confirmed vulnerability |
| **AC-Path** | Lateral movement from active session | active C2 session |
| **AC-Forge** | Infrastructure: VPS, tunnels, SSH keys | alongside AC-Echo |
| **AC-Quill** | Final report generation | goal achieved or deadlock |
| **AC-Strategist** | Attack path design | path unclear |

Each agent has a 3-layer boundary control: Supervisor pre-conditions → agent pre-flight gate → circuit breaker.

## Project Structure

```
AdversaryChefAgentTeam/
├── cmd/
│   ├── acactl/              # One-shot CLI: builds + starts services, dispatches tasks
│   │   ├── main.go
│   │   ├── commands/        # run, logs, project subcommands
│   │   ├── display/         # Terminal output formatting
│   │   └── lifecycle/       # Service lifecycle (build, start, stop)
│   └── acasched/            # Scheduler daemon (:9090)
│       ├── main.go
│       └── internal/
│           ├── api/         # REST API (projects, tasks, events)
│           ├── goose/       # Docker runner (goose containers)
│           ├── scheduler/   # Dispatcher, event loop, trigger, fallback poll
│           └── store/       # SQLite task/project persistence
├── servers/
│   ├── nexus/               # Graph DB + MCP server (:8081)
│   │   ├── cmd/server/
│   │   └── internal/
│   │       ├── models/      # Graph node types (Host, Service, Endpoint, etc.)
│   │       ├── store/       # SQLite store + EventedStore webhook decorator
│   │       └── tools/       # MCP tools (CRUD, graph query, scheduler bridge)
│   ├── kali/                # Async shell execution MCP server (:8080)
│   │   └── internal/
│   │       ├── job/         # Job manager (async exec, status tracking)
│   │       └── tools/       # exec, list_jobs, get_job, kill_job
│   └── mythic/              # Mythic C2 proxy MCP server (:8082)
│       └── internal/
│           ├── client/      # Mythic HTTP client
│           └── tools/       # callback, task, file, payload tools
├── prompts/
│   ├── red-team/            # Agent prompt files (.md)
│   │   ├── supervisor.md    # Read-only coordinator + dispatch pre-conditions
│   │   ├── echo-recon.md    # Fire-and-forget recon
│   │   ├── breach-exploit.md
│   │   ├── ghost-mythic.md
│   │   ├── path-lateral.md
│   │   ├── forge-resource.md
│   │   ├── quill-report.md
│   │   ├── strategist.md
│   │   └── squad.md         # Squad manifest
│   ├── _mcp-registry.yaml   # MCP server URL registry
│   ├── _squads.yaml         # Squad definitions
│   └── _tools/              # Tool documentation snippets
├── skills/
│   ├── _shared/scheduler/   # scheduler_* tool descriptions
│   └── red-team/kali/       # Kali tool descriptions
├── pkg/mcputil/              # Shared MCP server library
├── docker/                   # Dockerfiles + compose
├── docs/
│   ├── superpowers/specs/   # Design specs
│   └── research/            # Architecture research notes
├── go.work                   # Go workspace (4 modules)
├── .env                      # LLM provider config (gitignored)
└── acactl                    # Prebuilt CLI binary (gitignored)
```

## Boundary Control (3-Layer Defense)

```
Layer 1: Supervisor pre-condition matrix
  └─ Before dispatching ANY agent, verify all prerequisites exist in nexus graph.
     Missing? → Don't dispatch. Record skip reason.

Layer 2: Agent pre-flight gate
  └─ Step 0 of every agent workflow. Query nexus for prerequisites.
     Missing? → scheduler_complete_task immediately. Don't cross boundaries.

Layer 3: Circuit breaker
  └─ Agent: 3 repeated failures → stop. 3 harvest cycles with 0 progress → stop.
     Supervisor: 3 evaluations with 0 graph changes → deadlock → dispatch AC-Quill.
```

## Quick Start

### Prerequisites

- Go 1.26+
- Docker
- LLM API key (OpenAI-compatible)

### Setup

```bash
git clone git@github.com:Rvn0xsy/AdversaryChefAgentTeam.git
cd AdversaryChefAgentTeam

# Configure LLM provider
cat > .env << EOF
GOOSE_PROVIDER=openai
GOOSE_MODEL=deepseek-v4-flash
OPENAI_API_KEY=sk-your-key
OPENAI_HOST=https://api.deepseek.com
OPENAI_BASE_PATH=v1/chat/completions
EOF

# Build goose Docker image
docker build -t goose -f docker/goose/Dockerfile .
```

### Run an engagement

```bash
./acactl run -goal "对这个服务器进行渗透：113.31.118.180"
```

acactl will: build acasched → start kali-mcp → start nexus-mcp → start acasched → create project → dispatch Supervisor → observe the squad at work.

### Monitor progress

```bash
# Follow a specific task
./acactl logs <task-id> --follow

# Follow by project name
./acactl logs "Penetration Test" --follow

# Direct SQLite inspection
sqlite3 ~/.aca/data/acasched.db "SELECT substr(id,-8), agent, status FROM tasks ORDER BY created_at;"
```

### Development

```bash
# Build all
go build ./...

# Run individual servers
cd servers/kali  && go run ./cmd/server -port 8080
cd servers/nexus && go run ./cmd/server -port 8081 -db /tmp/nexus.db
cd cmd/acasched && go run . -db /tmp/acasched.db -prompts prompts -skills skills -env .env
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full development workflow: worktree → branch → PR → review → merge.

## Documents

- [Event-Driven Parallel Squad Design](docs/superpowers/specs/2026-07-26-event-driven-parallel-squad-design.md)
- [Agent Boundary Control Design](docs/superpowers/specs/2026-07-26-agent-boundary-control-design.md)
- [MCP Server Development Guide](docs/mcp-server-guide.md)
