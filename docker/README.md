# Docker Deployment

Containerized services for the AdversaryChefAgentTeam platform.

## Services

| Service | Port | Description |
|---------|------|-------------|
| `daemon` | — | Codex runtime executor, connects to all MCPs |
| `kali-mcp` | 8080 | Kali Linux toolchain (nmap, sqlmap, metasploit, etc.) |
| `asset-mcp` | 8081 | Asset/credential/clue/worklog CRUD, SQLite-backed |
| `mythic-mcp` | 8082 | Mythic C2 client proxy |

All MCP services expose a `/health` endpoint and communicate over the internal Docker network. No ports are exposed to the host by default.

## Directory Layout

```
docker/
├── docker-compose.yml            # Service orchestration
├── data/
│   ├── codex/                    # Codex configuration (mounted to /root/.codex)
│   ├── multica/                  # Multica templates (mounted to /root/.multica)
│   └── db/                       # asset-mcp SQLite data (mounted to /data)
├── daemon/Dockerfile
├── kali-mcp/Dockerfile
├── asset-mcp/Dockerfile
└── mythic-mcp/Dockerfile
```

## Quick Start

### Prerequisites

- Docker & Docker Compose
- A running Mythic C2 server (for mythic-mcp)

### 1. Configure

```bash
# Copy example configs
cp docker/data/codex/config.toml.example docker/data/codex/config.toml
cp docker/data/multica/docker-compose.selfhost.yml.example docker/data/multica/docker-compose.selfhost.yml
```

Edit `docker/data/codex/config.toml` with your cc-switch URL and API key.

### 2. Set environment variables

Create `docker/.env`:

```bash
MULTICA_API_URL=http://multica-server:3000
MYTHIC_SERVER=https://mythic.lab:7443
MYTHIC_API_KEY=your-api-token
```

### 3. Build & Run

```bash
cd docker
docker compose up -d --build
```

### 4. Verify

```bash
# Health checks
curl http://localhost:8080/health  # kali-mcp
curl http://localhost:8081/health  # asset-mcp
curl http://localhost:8082/health  # mythic-mcp (requires port mapping in override)

# Or check via docker
docker compose ps
docker compose logs -f daemon
```

## Debugging

To expose MCP ports to the host for debugging, create a `docker/docker-compose.override.yml`:

```yaml
services:
  kali-mcp:
    ports: ["8080:8080"]
  asset-mcp:
    ports: ["8081:8081"]
  mythic-mcp:
    ports: ["8082:8082"]
```

## Notes

- `kali-mcp` run pentesting tools inside the container
- `asset-mcp` persists SQLite data to `docker/data/db/`
- `mythic-mcp` is stateless — it proxies requests to an external Mythic server
