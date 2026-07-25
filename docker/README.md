# Docker Deployment

Containerized services for the AdversaryChefAgentTeam platform.

## Services

| Service | Port | Description |
|---------|------|-------------|
| `kali-mcp` | 8080 | Kali Linux toolchain (nmap, sqlmap, nuclei, etc.) |
| `nexus-mcp` | 8081 | Asset/credential/clue/worklog CRUD, SQLite-backed |
| `mythic-mcp` | 8082 | Mythic C2 client proxy |
| `acasched` | 9090 | Task scheduler, dispatches goose agents via MCPs |

All MCP services expose a `/health` endpoint and communicate over the internal Docker network.

## Directory Layout

```
docker/
├── docker-compose.yml            # Service orchestration
├── kali-mcp/Dockerfile
├── nexus-mcp/Dockerfile
├── mythic-mcp/Dockerfile
└── acasched/Dockerfile
```

## Quick Start

### Prerequisites

- Docker & Docker Compose
- A running Mythic C2 server (for mythic-mcp)

### 1. Set environment variables

Create `docker/.env`:

```bash
MYTHIC_SERVER=https://mythic.lab:7443
MYTHIC_API_KEY=your-api-token
```

### 2. Build & Run

```bash
cd docker
docker compose up -d --build
```

### 3. Verify

```bash
# Health checks
curl http://localhost:8080/health  # kali-mcp
curl http://localhost:8081/health  # nexus-mcp
curl http://localhost:8082/health  # mythic-mcp
curl http://localhost:9090/health  # acasched

# Or check via docker
docker compose ps
docker compose logs -f acasched
```

## Debugging

To expose MCP ports to the host for debugging, create a `docker/docker-compose.override.yml`:

```yaml
services:
  kali-mcp:
    ports: ["8080:8080"]
  nexus-mcp:
    ports: ["8081:8081"]
  mythic-mcp:
    ports: ["8082:8082"]
```

## Notes

- `kali-mcp` runs pentesting tools inside the container
- `nexus-mcp` persists SQLite data to a Docker volume
- `acasched` reads prompts from `../prompts` (mounted read-only)
- `mythic-mcp` is stateless — it proxies requests to an external Mythic server
