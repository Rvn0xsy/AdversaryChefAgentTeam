# AdversaryChefAgentTeam

Red team agent platform — collaborative penetration testing with Multica + Codex + MCP.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Multica Server                         │
│               (task orchestration · issue tracking)          │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│                    Daemon + Codex CLI                        │
│          (runtime executor · cc-switch → model)              │
└────┬────────────────┬──────────────────┬────────────────────┘
     │                │                  │
┌────▼────┐    ┌──────▼──────┐    ┌──────▼──────┐
│ kali-mcp│    │  asset-mcp  │    │  mythic-mcp │
│  :8080  │    │   :8081     │    │   :8082     │
│ Kali    │    │ SQLite CRUD │    │ Mythic C2   │
│ tools   │    │ assets/creds│    │ proxy       │
└─────────┘    └─────────────┘    └─────────────┘
```

## Project Structure

```
AdversaryChefAgentTeam/
├── go.work                       # Go monorepo workspace (4 modules)
├── pkg/mcputil/                  # Shared MCP server library
│   └── mcputil.go                # ServerConfig · TextResult · Run() · /health · middleware
├── servers/
│   ├── kali/                     # Kali MCP server (port 8080)
│   │   ├── cmd/server/main.go
│   │   └── internal/{job,tools}/
│   ├── asset/                    # Asset MCP server (port 8081)
│   │   ├── cmd/server/main.go
│   │   └── internal/{models,store,tools}/
│   └── mythic/                   # Mythic C2 proxy server (port 8082)
│       ├── cmd/server/main.go
│       └── internal/{client,tools}/
├── docker/                       # Docker Compose + container definitions
│   ├── docker-compose.yml
│   ├── README.md
│   ├── data/{codex,multica,db}/  # Persistent data & config templates
│   ├── daemon/Dockerfile
│   ├── kali-mcp/Dockerfile
│   ├── asset-mcp/Dockerfile
│   └── mythic-mcp/Dockerfile
├── skills/                       # Red team playbooks (SKILL.md skeletons)
│   ├── penetration-methodology/
│   ├── owasp-checklist/
│   ├── compliance-rules/
│   ├── weekly-report-template/
│   └── retrospective-template/
├── docs/                         # Design docs · architecture decisions
└── scripts/                      # Setup & verification scripts
```

## MCP Servers

| Server | Port | Module | Description |
|--------|------|--------|-------------|
| **kali** | 8080 | `adversarychef/kali` | Async job execution: nmap, sqlmap, metasploit, gobuster, hydra |
| **asset** | 8081 | `adversarychef/asset` | CRUD for projects, assets, clues, credentials, work logs (SQLite) |
| **mythic** | 8082 | `adversarychef/mythic` | Mythic C2 proxy: callbacks, tasks, files, payloads |

Shared library `pkg/mcputil` provides:
- `ServerConfig` — unified flag parsing (`--host`, `--port`, `--db`, `--mythic-server`, etc.)
- `TextResult()` — MCP text response helper
- `Run()` — SSE handler + `/health` + request logging + panic recovery + graceful shutdown

## Quick Start

### Prerequisites

- Docker & Docker Compose v2
- Multica server (self-hosted)
- Codex CLI with cc-switch configured
- (Optional) Mythic C2 server for mythic-mcp

### Setup

```bash
# Clone
git clone <repo-url> && cd AdversaryChefAgentTeam

# Configure
cp docker/data/codex/config.toml.example docker/data/codex/config.toml
cp docker/data/multica/docker-compose.selfhost.yml.example docker/data/multica/docker-compose.selfhost.yml
# Edit config.toml with your cc-switch URL and API key

# Set environment
cat > docker/.env << EOF
MULTICA_API_URL=http://multica-server:3000
MYTHIC_SERVER=https://mythic.lab:7443
MYTHIC_API_KEY=your-token
EOF

# Build & run
cd docker && docker compose up -d --build
```

### Local Development

```bash
# Build all Go modules
go work sync && go build ./...

# Run individual servers
cd servers/kali  && go run ./cmd/server  -port 8080
cd servers/asset && go run ./cmd/server  -port 8081 -db /tmp/asset.db
cd servers/mythic && go run ./cmd/server -port 8082 -mythic-server https://mythic.lab:7443 -mythic-api-key YOUR_KEY

# Run tests
go test -C servers/kali ./...
```

## Technology

- **Go 1.26.4** — all MCP servers
- **[mcp-go-sdk](https://github.com/modelcontextprotocol/go-sdk) v1.6.1** — MCP protocol
- **HTTP/SSE transport** — not stdio
- **go.work monorepo** — 4 modules, one workspace
- **Docker Compose** — orchestrated deployment, no exposed ports, all bind-mount volumes

## Documents

- [Technical Design](docs/red-team-agent-design.md)
- [MCP Server Development Guide](docs/mcp-server-guide.md)
- [Phase 0 Experiment Notes](docs/phase-0-notes.md)
- [Docker Deployment](docker/README.md)
