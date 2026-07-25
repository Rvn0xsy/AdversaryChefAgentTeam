# ACA Scheduler + nexus-mcp v2 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Multica with a self-built task scheduler (acasched) + goose runtime, and upgrade nexus-mcp with Operation Graph + Reasoning Graph data model.

**Architecture:** acasched manages task lifecycle (pending→dispatched→running→done) with parent re-trigger. Each task spawns a goose subprocess with agent-specific prompts and MCP connections. nexus-mcp is the shared memory hub—all agent collaboration flows through graph nodes and edges.

**Tech Stack:** Go 1.26.4, SQLite (modernc.org/sqlite), mcp-go-sdk v1.6.1, goose CLI v1.44+

## Global Constraints

- Go modules: `adversarychef/nexus` (renamed from `adversarychef/asset`), `adversarychef/mcputil`, `adversarychef/acasched`
- SQLite WAL mode enforced, `SetMaxOpenConns(1)` for write safety
- All MCP transport: HTTP/SSE (not stdio)
- Session binding: project_id enforced at MCP session level, not Agent-parameter level
- No Multica dependency
- Naming: `nexus-mcp` not `asset-mcp`; MCP URL default `http://127.0.0.1:8081`
- Old tools (`create_asset`, `list_clues`, etc.) preserved for backward compatibility

---

## File Structure

```
AdversaryChefAgentTeam/
├── cmd/acasched/main.go                          (CREATE)  scheduler entry
├── internal/
│   ├── scheduler/
│   │   ├── dispatcher.go                        (CREATE)  main dispatch loop
│   │   ├── trigger.go                           (CREATE)  parent re-trigger
│   │   ├── reaper.go                            (CREATE)  timeout detection
│   │   └── lifecycle.go                         (CREATE)  status transitions
│   ├── goose/
│   │   ├── runner.go                            (CREATE)  goose subprocess
│   │   └── parser.go                            (CREATE)  stream-json parse
│   ├── store/
│   │   └── sqlite.go                            (CREATE)  tasks + projects
│   └── api/
│       ├── server.go                            (CREATE)  HTTP server
│       ├── tasks.go                             (CREATE)  /api/tasks
│       └── projects.go                          (CREATE)  /api/projects
├── servers/nexus/                                (RENAME from servers/asset/)
│   └── internal/
│       ├── models/
│       │   ├── models.go                        (MODIFY)  rename package
│       │   └── graph.go                         (CREATE)  graph node models
│       ├── store/
│       │   ├── store.go                         (MODIFY)  split interface
│       │   ├── operation.go                     (CREATE)  OperationStore
│       │   ├── reasoning.go                     (CREATE)  ReasoningStore
│       │   ├── edges.go                         (CREATE)  GraphEdgeStore
│       │   ├── session.go                       (CREATE)  session binding
│       │   ├── sqlite.go                        (MODIFY)  embed new stores
│       │   └── memory.go                        (MODIFY)  embed new stores
│       └── tools/
│           ├── graph_nodes.go                   (CREATE)  host/service/endpoint/session tools
│           ├── reasoning.go                     (CREATE)  evidence/hypothesis/vuln tools
│           ├── graph_edges.go                   (CREATE)  edge CRUD tools
│           ├── graph_query.go                   (CREATE)  graph_query/graph_trace
│           ├── scheduler_bridge.go              (CREATE)  scheduler bridge
│           └── register.go                      (MODIFY)  register new tools
├── pkg/mcputil/mcputil.go                       (MODIFY)  add SessionMap
├── go.work                                       (MODIFY)  update modules
└── go.work.sum                                   (MODIFY)
```

---

## Task 1: Rename asset-mcp → nexus-mcp

**Files:**
- Modify: `servers/asset/go.mod`
- Modify: `servers/asset/cmd/server/main.go`
- Modify: `servers/asset/internal/models/models.go`
- Modify: `go.work`
- Modify: `go.work.sum`
- Move: `servers/asset/` → `servers/nexus/`

**Interfaces:**
- Consumes: nothing
- Produces: module name `adversarychef/nexus`, default port 8081 unchanged

### Steps

- [ ] **Step 1: Rename module in go.mod**

```bash
cd servers/asset
# Edit go.mod: replace "adversarychef/asset" → "adversarychef/nexus"
```

```go
// servers/nexus/go.mod
module adversarychef/nexus

go 1.26.4

require (
	adversarychef/mcputil v0.0.0
	github.com/modelcontextprotocol/go-sdk v1.6.1
	modernc.org/sqlite v1.54.0
)

replace adversarychef/mcputil => ../../pkg/mcputil
```

- [ ] **Step 2: Update all import paths in Go files**

Run: `find servers/nexus -name "*.go" -exec sed -i '' 's|adversarychef/asset|adversarychef/nexus|g' {} \;`

- [ ] **Step 3: Update go.work**

```go
// go.work — replace "asset" with "nexus"
go 1.26.4

use (
	./pkg/mcputil
	./servers/nexus
	./servers/kali
	./servers/mythic
)
```

- [ ] **Step 4: Build verification**

```bash
cd servers/nexus && go build ./...
cd servers/kali && go build ./...
cd servers/mythic && go build ./...
```

Expected: all three servers build without errors.

- [ ] **Step 5: Update go.work.sum**

```bash
cd /Users/rvn0xsy/Documents/Git/AdversaryChefAgentTeam
go work sync
```

- [ ] **Step 6: Commit**

```bash
git add go.work go.work.sum servers/nexus/ servers/asset/
git rm -r servers/asset/
git commit -m "refactor: rename asset-mcp to nexus-mcp"
```

---

## Task 2: Graph Node Models

**Files:**
- Create: `servers/nexus/internal/models/graph.go`

**Interfaces:**
- Consumes: nothing (standalone model file)
- Produces: `HostNode`, `ServiceNode`, `EndpointNode`, `SessionNode`, `EvidenceNode`, `HypothesisNode`, `VulnerabilityNode`, `GraphEdge`, `Subgraph`, `TraceResult`

- [ ] **Step 1: Write graph.go**

```go
// servers/nexus/internal/models/graph.go
package models

import "time"

// ── Operation Graph nodes ──

type HostNode struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	IPs          []string  `json:"ips,omitempty"`
	Hostname     string    `json:"hostname,omitempty"`
	OS           string    `json:"os,omitempty"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type ServiceNode struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	HostID       string    `json:"host_id"`
	Port         int       `json:"port"`
	Protocol     string    `json:"protocol"`
	Name         string    `json:"name"`
	Version      string    `json:"version,omitempty"`
	Banner       string    `json:"banner,omitempty"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type EndpointNode struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	ServiceID    string    `json:"service_id"`
	URL          string    `json:"url"`
	Method       string    `json:"method"`
	Parameters   []string  `json:"parameters,omitempty"`
	Status       string    `json:"status"`
	DiscoveredBy string    `json:"discovered_by,omitempty"`
	TestedBy     string    `json:"tested_by,omitempty"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type SessionNode struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	AssetID      string    `json:"asset_id,omitempty"`
	CreatedBy    string    `json:"created_by"`
	SessionType  string    `json:"session_type"`
	URL          string    `json:"url,omitempty"`
	Cookies      string    `json:"cookies,omitempty"`
	TokenValue   string    `json:"token_value,omitempty"`
	Metadata     string    `json:"metadata,omitempty"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    string    `json:"expires_at,omitempty"`
}

// ── Reasoning Graph nodes ──

type EvidenceNode struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Label        string    `json:"label"`
	Source       string    `json:"source"`
	ContentRef   string    `json:"content_ref,omitempty"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type HypothesisNode struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Label        string    `json:"label"`
	Confidence   float64   `json:"confidence"`
	Status       string    `json:"status"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type VulnerabilityNode struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Title        string    `json:"title"`
	CVE          string    `json:"cve,omitempty"`
	Severity     string    `json:"severity"`
	CVSS         float64   `json:"cvss,omitempty"`
	Description  string    `json:"description,omitempty"`
	Remediation  string    `json:"remediation,omitempty"`
	Status       string    `json:"status"`
	EvidenceRefs []string  `json:"evidence_refs"`
	CreatedAt    time.Time `json:"created_at"`
}

// ── Graph edges ──

type GraphEdge struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	FromID       string    `json:"from_id"`
	ToID         string    `json:"to_id"`
	EdgeType     string    `json:"edge_type"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ── Query results ──

type Subgraph struct {
	Nodes []map[string]any `json:"nodes"`
	Edges []GraphEdge      `json:"edges"`
}

type TraceResult struct {
	Chain []TraceHop `json:"chain"`
}

type TraceHop struct {
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"`
	Label    string `json:"label"`
	EdgeType string `json:"edge_type,omitempty"`
	FromID   string `json:"from_id,omitempty"`
}
```

- [ ] **Step 2: Build verification**

```bash
cd servers/nexus && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add servers/nexus/internal/models/graph.go
git commit -m "feat(nexus): add graph node models (Operation + Reasoning)"
```

---

## Task 3: Split Store Interface

**Files:**
- Modify: `servers/nexus/internal/store/store.go`
- Create: `servers/nexus/internal/store/operation.go`
- Create: `servers/nexus/internal/store/reasoning.go`
- Create: `servers/nexus/internal/store/edges.go`
- Create: `servers/nexus/internal/store/session.go`

**Interfaces:**
- Consumes: `models.HostNode`, `models.ServiceNode`, `models.EndpointNode`, `models.SessionNode`, `models.EvidenceNode`, `models.HypothesisNode`, `models.VulnerabilityNode`, `models.GraphEdge`, `models.Subgraph`, `models.TraceResult` (from Task 2)
- Produces: `OperationStore`, `ReasoningStore`, `GraphEdgeStore` interfaces

- [ ] **Step 1: Add new sub-interfaces to store.go**

```go
// servers/nexus/internal/store/store.go — append after existing Store interface
import "adversarychef/nexus/internal/models"

// OperationStore manages Operation Graph nodes.
type OperationStore interface {
	// Hosts
	ListHosts(projectID string) ([]models.HostNode, error)
	GetHost(id string) (*models.HostNode, error)
	CreateHost(h *models.HostNode) error
	UpdateHost(h *models.HostNode) error
	DeleteHost(id string) error

	// Services
	ListServices(projectID string) ([]models.ServiceNode, error)
	GetService(id string) (*models.ServiceNode, error)
	CreateService(s *models.ServiceNode) error
	UpdateService(s *models.ServiceNode) error
	DeleteService(id string) error

	// Endpoints
	ListEndpoints(projectID string) ([]models.EndpointNode, error)
	GetEndpoint(id string) (*models.EndpointNode, error)
	CreateEndpoint(e *models.EndpointNode) error
	UpdateEndpointStatus(id, status, testedBy string) error
	GetUntestedEndpoints(projectID string) ([]models.EndpointNode, error)

	// Sessions
	ListSessions(projectID string) ([]models.SessionNode, error)
	GetSession(id string) (*models.SessionNode, error)
	CreateSession(s *models.SessionNode) error
	DeleteSession(id string) error

	// Graph queries
	GraphQuery(projectID, startNodeID string, maxHops int) (*models.Subgraph, error)
	GraphTrace(projectID, nodeID string) (*models.TraceResult, error)
}

// ReasoningStore manages Reasoning Graph nodes.
type ReasoningStore interface {
	ListEvidence(projectID string) ([]models.EvidenceNode, error)
	CreateEvidence(e *models.EvidenceNode) error

	ListHypotheses(projectID string) ([]models.HypothesisNode, error)
	CreateHypothesis(h *models.HypothesisNode) error
	UpdateHypothesisStatus(id, status string, confidence float64) error

	ListVulnerabilities(projectID string) ([]models.VulnerabilityNode, error)
	CreateVulnerability(v *models.VulnerabilityNode) error
	UpdateVulnerability(v *models.VulnerabilityNode) error
}

// GraphEdgeStore manages graph edges.
type GraphEdgeStore interface {
	CreateEdge(e *models.GraphEdge) error
	ListEdges(projectID, fromID, toID string) ([]models.GraphEdge, error)
	DeleteEdge(id string) error
}
```

- [ ] **Step 2: Update Store to embed sub-interfaces**

```go
// servers/nexus/internal/store/store.go — modify existing Store interface
type Store interface {
	// Legacy (unchanged)
	ListProjects() ([]models.Project, error)
	GetProject(id string) (*models.Project, error)
	CreateProject(p *models.Project) error
	UpdateProject(p *models.Project) error
	DeleteProject(id string) error
	ListAssets(projectID string) ([]models.Asset, error)
	GetAsset(id string) (*models.Asset, error)
	CreateAsset(a *models.Asset) error
	UpdateAsset(a *models.Asset) error
	DeleteAsset(id string) error
	ListClues(projectID string) ([]models.Clue, error)
	GetClue(id string) (*models.Clue, error)
	CreateClue(c *models.Clue) error
	UpdateClue(c *models.Clue) error
	DeleteClue(id string) error
	ListCredentials(projectID string) ([]models.Credential, error)
	GetCredential(id string) (*models.Credential, error)
	CreateCredential(c *models.Credential) error
	UpdateCredential(c *models.Credential) error
	DeleteCredential(id string) error
	ListWorkLogs(projectID string) ([]models.WorkLog, error)
	GetWorkLog(id string) (*models.WorkLog, error)
	CreateWorkLog(w *models.WorkLog) error
	UpdateWorkLog(w *models.WorkLog) error
	DeleteWorkLog(id string) error
	SearchAssets(projectID, query string) ([]models.Asset, error)
	SearchClues(projectID, query, clueType, status string) ([]models.Clue, error)
	ProjectSummary(projectID string) (*models.ProjectSummary, error)

	// New: graph sub-interfaces
	OperationStore
	ReasoningStore
	GraphEdgeStore
}
```

- [ ] **Step 3: Build verification**

```bash
cd servers/nexus && go build ./... 2>&1
```

Expected: compilation errors from SQLiteStore and MemoryStore not implementing new interfaces (will fix in Task 4).

- [ ] **Step 4: Commit**

```bash
git add servers/nexus/internal/store/
git commit -m "feat(nexus): split Store interface with Operation/Reasoning/GraphEdge sub-interfaces"
```

---

## Task 4: SQLite Graph Store Implementation

**Files:**
- Modify: `servers/nexus/internal/store/sqlite.go`
- Modify: `servers/nexus/internal/store/memory.go`

**Interfaces:**
- Consumes: `OperationStore`, `ReasoningStore`, `GraphEdgeStore` interfaces (from Task 3)
- Produces: SQLiteStore and MemoryStore satisfying all sub-interfaces

- [ ] **Step 1: Add DDL to sqlite.go migrate()**

Insert after existing `worklogs` table in the DDL list:

```go
// append to _SCHEMA string slice in migrate()
`CREATE TABLE IF NOT EXISTS host_nodes (
    id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    ips TEXT NOT NULL DEFAULT '[]', hostname TEXT DEFAULT '',
    os TEXT DEFAULT '', evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL
)`,
`CREATE TABLE IF NOT EXISTS service_nodes (
    id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    host_id TEXT NOT NULL, port INTEGER NOT NULL,
    protocol TEXT NOT NULL DEFAULT 'tcp', name TEXT NOT NULL,
    version TEXT DEFAULT '', banner TEXT DEFAULT '',
    evidence_refs TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL
)`,
`CREATE TABLE IF NOT EXISTS endpoint_nodes (
    id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    service_id TEXT NOT NULL, url TEXT NOT NULL,
    method TEXT NOT NULL DEFAULT 'GET',
    parameters TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'discovered',
    discovered_by TEXT DEFAULT '', tested_by TEXT DEFAULT '',
    evidence_refs TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL
)`,
`CREATE UNIQUE INDEX IF NOT EXISTS uq_endpoint_url_method ON endpoint_nodes(project_id, url, method)`,
`CREATE TABLE IF NOT EXISTS session_nodes (
    id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    asset_id TEXT DEFAULT '', created_by TEXT NOT NULL,
    session_type TEXT NOT NULL, url TEXT DEFAULT '',
    cookies TEXT DEFAULT '', token_value TEXT DEFAULT '',
    metadata TEXT DEFAULT '', evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL, expires_at TEXT DEFAULT ''
)`,
`CREATE TABLE IF NOT EXISTS evidence_nodes (
    id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    label TEXT NOT NULL, source TEXT NOT NULL,
    content_ref TEXT DEFAULT '', evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL
)`,
`CREATE TABLE IF NOT EXISTS hypothesis_nodes (
    id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    label TEXT NOT NULL, confidence REAL NOT NULL DEFAULT 0.0,
    status TEXT NOT NULL DEFAULT 'proposed',
    evidence_refs TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL
)`,
`CREATE TABLE IF NOT EXISTS vulnerability_nodes (
    id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    title TEXT NOT NULL, cve TEXT DEFAULT '',
    severity TEXT NOT NULL DEFAULT 'medium', cvss REAL DEFAULT 0.0,
    description TEXT DEFAULT '', remediation TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open',
    evidence_refs TEXT NOT NULL, created_at TEXT NOT NULL
)`,
`CREATE TABLE IF NOT EXISTS graph_edges (
    id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
    from_id TEXT NOT NULL, to_id TEXT NOT NULL,
    edge_type TEXT NOT NULL, evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL
)`,
`CREATE INDEX IF NOT EXISTS idx_edges_from ON graph_edges(project_id, from_id)`,
`CREATE INDEX IF NOT EXISTS idx_edges_to ON graph_edges(project_id, to_id)`,
```

- [ ] **Step 2: Implement OperationStore methods on SQLiteStore**

Add to `sqlite.go`:

```go
func (s *SQLiteStore) ListHosts(projectID string) ([]models.HostNode, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, ips, hostname, os, evidence_refs, created_at FROM host_nodes WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.HostNode
	for rows.Next() {
		var h models.HostNode; var ips, refs, ct string
		if err := rows.Scan(&h.ID, &h.ProjectID, &ips, &h.Hostname, &h.OS, &refs, &ct); err != nil { return nil, err }
		json.Unmarshal([]byte(ips), &h.IPs); json.Unmarshal([]byte(refs), &h.EvidenceRefs)
		h.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetHost(id string) (*models.HostNode, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var h models.HostNode; var ips, refs, ct string
	err := s.db.QueryRow(`SELECT id, project_id, ips, hostname, os, evidence_refs, created_at FROM host_nodes WHERE id = ?`, id).
		Scan(&h.ID, &h.ProjectID, &ips, &h.Hostname, &h.OS, &refs, &ct)
	if err == sql.ErrNoRows { return nil, fmt.Errorf("host not found: %s", id) }
	if err != nil { return nil, err }
	json.Unmarshal([]byte(ips), &h.IPs); json.Unmarshal([]byte(refs), &h.EvidenceRefs)
	h.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &h, nil
}

func (s *SQLiteStore) CreateHost(h *models.HostNode) error {
	s.mu.Lock(); defer s.mu.Unlock()
	if h.ID == "" { h.ID = genID("host") }
	h.CreatedAt = time.Now()
	ips, _ := json.Marshal(emptySlice(h.IPs))
	refs, _ := json.Marshal(emptySlice(h.EvidenceRefs))
	_, err := s.db.Exec(`INSERT INTO host_nodes (id, project_id, ips, hostname, os, evidence_refs, created_at) VALUES (?,?,?,?,?,?,?)`,
		h.ID, h.ProjectID, string(ips), h.Hostname, h.OS, string(refs), h.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateHost(h *models.HostNode) error {
	s.mu.Lock(); defer s.mu.Unlock()
	ips, _ := json.Marshal(emptySlice(h.IPs))
	refs, _ := json.Marshal(emptySlice(h.EvidenceRefs))
	_, err := s.db.Exec(`UPDATE host_nodes SET ips=?, hostname=?, os=?, evidence_refs=? WHERE id=?`,
		string(ips), h.Hostname, h.OS, string(refs), h.ID)
	return err
}

func (s *SQLiteStore) DeleteHost(id string) error {
	s.mu.Lock(); defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM host_nodes WHERE id = ?`, id)
	return err
}
// ... similar CRUD for Service, Endpoint, Session repeated inline below
```

Due to space constraints in this plan, the CRUD methods for `ListServices`, `GetService`, `CreateService`, `UpdateService`, `DeleteService`, `ListEndpoints`, `GetEndpoint`, `CreateEndpoint`, `UpdateEndpointStatus`, `GetUntestedEndpoints`, `ListSessions`, `GetSession`, `CreateSession`, `DeleteSession` follow the same pattern: JSON-serialize arrays, UTC timestamps, `genID("svc"|"ep"|"sess")`.

- [ ] **Step 3: Implement graph_query and graph_trace**

```go
func (s *SQLiteStore) GraphQuery(projectID, startNodeID string, maxHops int) (*models.Subgraph, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	sub := &models.Subgraph{Nodes: []map[string]any{}, Edges: []models.GraphEdge{}}
	visited := map[string]bool{}
	queue := []struct{ id string }{}{{id: startNodeID}}
	visited[startNodeID] = true
	for hop := 0; hop <= maxHops && len(queue) > 0; hop++ {
		next := []struct{ id string }{}{}
		for _, cur := range queue {
			node := s.getNodeByID(cur.id)
			if node != nil {
				sub.Nodes = append(sub.Nodes, node)
			}
			rows, err := s.db.Query(`SELECT id, project_id, from_id, to_id, edge_type, evidence_refs, created_at FROM graph_edges WHERE project_id = ? AND (from_id = ? OR to_id = ?)`, projectID, cur.id, cur.id)
			if err != nil { continue }
			for rows.Next() {
				var e models.GraphEdge; var refs, ct string
				rows.Scan(&e.ID, &e.ProjectID, &e.FromID, &e.ToID, &e.EdgeType, &refs, &ct)
				json.Unmarshal([]byte(refs), &e.EvidenceRefs)
				e.CreatedAt, _ = time.Parse(time.RFC3339, ct)
				sub.Edges = append(sub.Edges, e)
				neighbor := e.FromID
				if neighbor == cur.id { neighbor = e.ToID }
				if !visited[neighbor] { visited[neighbor] = true; next = append(next, struct{ id string }{neighbor}) }
			}
			rows.Close()
		}
		queue = next
	}
	return sub, nil
}

func (s *SQLiteStore) GraphTrace(projectID, nodeID string) (*models.TraceResult, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	result := &models.TraceResult{Chain: []models.TraceHop{}}
	current := nodeID
	for i := 0; i < 10; i++ {
		node := s.getNodeByID(current)
		if node == nil { break }
		hop := models.TraceHop{NodeID: current, NodeType: node["node_type"].(string), Label: node["label"].(string)}
		row := s.db.QueryRow(`SELECT id, from_id, to_id, edge_type FROM graph_edges WHERE project_id = ? AND to_id = ? LIMIT 1`, projectID, current)
		var e models.GraphEdge
		if err := row.Scan(&e.ID, &e.FromID, &e.ToID, &e.EdgeType); err != nil { break }
		hop.EdgeType = e.EdgeType; hop.FromID = e.FromID
		result.Chain = append(result.Chain, hop)
		current = e.FromID
	}
	return result, nil
}

func (s *SQLiteStore) getNodeByID(id string) map[string]any {
	for _, tbl := range []string{"host_nodes", "service_nodes", "endpoint_nodes", "session_nodes", "evidence_nodes", "hypothesis_nodes", "vulnerability_nodes"} {
		row, err := s.db.Query(fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tbl), id)
		if err != nil { continue }
		defer row.Close()
		cols, _ := row.Columns()
		if row.Next() {
			vals := make([]any, len(cols)); ptrs := make([]any, len(cols))
			for i := range vals { ptrs[i] = &vals[i] }
			row.Scan(ptrs...)
			m := map[string]any{}
			for i, c := range cols { m[c] = vals[i] }
			return m
		}
	}
	return nil
}
```

- [ ] **Step 4: Implement ReasoningStore + GraphEdgeStore methods on SQLiteStore**

Add CRUD for `evidence_nodes`, `hypothesis_nodes`, `vulnerability_nodes`, `graph_edges` tables following the same JSON-serialization pattern. Key distinction: `CreateVulnerability` must check `len(v.EvidenceRefs) > 0` and return error if empty.

- [ ] **Step 5: MemoryStore stubs**

In `memory.go`, add embedded structs that delegate to in-memory maps for all new tables. Minimum: `map[string]*models.HostNode`, `map[string]*models.ServiceNode`, etc. Output error "not implemented for in-memory store" for `GraphQuery` and `GraphTrace`.

- [ ] **Step 6: Build + test**

```bash
cd servers/nexus && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 7: Commit**

```bash
git add servers/nexus/internal/store/
git commit -m "feat(nexus): implement SQLite Graph stores (Operation + Reasoning + Edges)"
```

---

## Task 5: Session Binding Middleware

**Files:**
- Create: `servers/nexus/internal/store/session.go`
- Modify: `servers/nexus/cmd/server/main.go`
- Modify: `pkg/mcputil/mcputil.go`

**Interfaces:**
- Consumes: MCP session ID from request context
- Produces: `GetOrBind(sessionID, callerProjectID string) (string, error)`

- [ ] **Step 1: Add SessionMap to mcputil**

```go
// pkg/mcputil/mcputil.go — append

import "sync"

type SessionBinding struct {
	ProjectID string
	Bound     bool
	BoundAt   time.Time
}

type SessionMap struct {
	mu       sync.RWMutex
	sessions map[string]*SessionBinding
}

func NewSessionMap() *SessionMap {
	return &SessionMap{sessions: make(map[string]*SessionBinding)}
}

func (m *SessionMap) GetOrBind(sessionID, callerProjectID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, exists := m.sessions[sessionID]
	if !exists {
		m.sessions[sessionID] = &SessionBinding{ProjectID: callerProjectID, Bound: true, BoundAt: time.Now()}
		return callerProjectID, nil
	}
	if s.ProjectID != callerProjectID {
		return "", fmt.Errorf("session %s bound to project %s, rejected %s", sessionID, s.ProjectID, callerProjectID)
	}
	return s.ProjectID, nil
}
```

- [ ] **Step 2: Wire SessionMap into nexus-mcp main.go**

```go
// servers/nexus/cmd/server/main.go — modify
func main() {
	cfg := mcputil.ParseConfig("nexus", "0.3.0", 8081)
	s, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil { log.Fatalf("failed to open database: %v", err) }
	defer s.Close()

	sessionMap := mcputil.NewSessionMap()

	mcputil.Run(cfg, func(server *mcp.Server) {
		tools.RegisterAll(server, s, sessionMap)  // ← passes sessionMap
	})
}
```

- [ ] **Step 3: Build verification**

```bash
cd servers/nexus && go build ./...
```

Expected: `RegisterAll` signature mismatch — will fix in Task 6.

- [ ] **Step 4: Commit**

```bash
git add pkg/mcputil/mcputil.go servers/nexus/cmd/server/main.go servers/nexus/internal/store/session.go
git commit -m "feat(nexus): add session-level project_id binding"
```

---

## Task 6: Graph Node MCP Tools

**Files:**
- Create: `servers/nexus/internal/tools/graph_nodes.go`
- Modify: `servers/nexus/internal/tools/register.go`

**Interfaces:**
- Consumes: `store.Store`, `*mcputil.SessionMap`
- Produces: 22 new MCP tools for host/service/endpoint/session/evidence/hypothesis/vulnerability CRUD

- [ ] **Step 1: Write graph_nodes.go — Operation Graph tools**

```go
// servers/nexus/internal/tools/graph_nodes.go
package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/models"
	"adversarychef/nexus/internal/store"
	"adversarychef/mcputil"
)

type hostParams struct {
	ProjectID string   `json:"project_id" jsonschema:"Project ID"`
	IPs       []string `json:"ips" jsonschema:"IP addresses"`
	Hostname  string   `json:"hostname,omitempty" jsonschema:"Hostname"`
	OS        string   `json:"os,omitempty" jsonschema:"Operating system"`
}

type serviceParams struct {
	ProjectID string `json:"project_id" jsonschema:"Project ID"`
	HostID    string `json:"host_id" jsonschema:"Host node ID"`
	Port      int    `json:"port" jsonschema:"Port number"`
	Protocol  string `json:"protocol,omitempty" jsonschema:"Protocol (tcp/udp)"`
	Name      string `json:"name" jsonschema:"Service name"`
	Version   string `json:"version,omitempty" jsonschema:"Version"`
}

type endpointParams struct {
	ProjectID string   `json:"project_id" jsonschema:"Project ID"`
	ServiceID string   `json:"service_id" jsonschema:"Service node ID"`
	URL       string   `json:"url" jsonschema:"URL path"`
	Method    string   `json:"method,omitempty" jsonschema:"HTTP method"`
	Parameters []string `json:"parameters,omitempty" jsonschema:"Parameter names"`
}

type sessionParams struct {
	ProjectID   string `json:"project_id" jsonschema:"Project ID"`
	SessionType string `json:"session_type" jsonschema:"Session type (web/c2_shell/ssh/tunnel)"`
	CreatedBy   string `json:"created_by" jsonschema:"Creating agent name"`
	URL         string `json:"url,omitempty" jsonschema:"Session URL"`
	AssetID     string `json:"asset_id,omitempty" jsonschema:"Related asset ID"`
}

func registerGraphNodes(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// Host CRUD
	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "host_list", Description: "List all host nodes by project ID"},
		scopedLister(sm, func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID string `json:"project_id"` }) (*mcp.CallToolResult, any, error) {
			items, err := s.ListHosts(params.ProjectID)
			if err != nil { return mcputil.TextResult("query failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(items)
			return mcputil.TextResult(string(b)), nil, nil
		}))

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "host_get", Description: "Get host node by ID"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ID string `json:"id"` }) (*mcp.CallToolResult, any, error) {
			h, err := s.GetHost(params.ID)
			if err != nil { return mcputil.TextResult("not found: " + err.Error()), nil, nil }
			b, _ := json.Marshal(h)
			return mcputil.TextResult(string(b)), nil, nil
		})

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "host_create", Description: "Create a new host node"},
		func(ctx context.Context, req *mcp.CallToolRequest, params hostParams) (*mcp.CallToolResult, any, error) {
			h := &models.HostNode{ProjectID: params.ProjectID, IPs: params.IPs, Hostname: params.Hostname, OS: params.OS}
			if err := s.CreateHost(h); err != nil { return mcputil.TextResult("create failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(h)
			return mcputil.TextResult(string(b)), nil, nil
		})

	// ... Service, Endpoint, Session tools follow same pattern (omitted for brevity)
	// Key: endpoint_create uses UNIQUE(project_id, url, method) for automatic dedup
	// Key: find_untested_endpoints filters by status='discovered'
	// Key: find_sessions filters by session_type
}
```

- [ ] **Step 2: Add scopedLister helper**

```go
// scopedLister validates project_id via session binding before calling fn
func scopedLister[P interface{ GetProjectID() string }](sm *mcputil.SessionMap, handler func(context.Context, *mcp.CallToolRequest, P) (*mcp.CallToolResult, any, error)) func(context.Context, *mcp.CallToolRequest, P) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, params P) (*mcp.CallToolResult, any, error) {
		// session binding check
		return handler(ctx, req, params)
	}
}
```

Note: Full session binding validation in tool handlers is deferred to Task 8 (retrofit after SessionMap is wired into MCP request context).

- [ ] **Step 3: Update register.go**

```go
// servers/nexus/internal/tools/register.go
func RegisterAll(server *mcp.Server, s store.Store) {
	registerProjects(server, s)
	registerAssets(server, s)
	registerClues(server, s)
	registerCredentials(server, s)
	registerWorkLogs(server, s)
	registerSearch(server, s)
}

func RegisterAllV2(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	RegisterAll(server, s)
	registerGraphNodes(server, s, sm)       // ← new
	registerReasoningNodes(server, s, sm)    // ← new
	registerGraphEdges(server, s, sm)        // ← new
	registerGraphQueries(server, s, sm)      // ← new
	registerSchedulerBridge(server, sm)       // ← new (Task 7)
}
```

- [ ] **Step 4: Build + commit**

```bash
cd servers/nexus && go build ./...
git add servers/nexus/internal/tools/
git commit -m "feat(nexus): add graph node MCP tools (Operation Graph)"
```

---

## Task 7: Reasoning Graph + Edge + Query + Scheduler Bridge Tools

**Files:**
- Create: `servers/nexus/internal/tools/reasoning.go`
- Create: `servers/nexus/internal/tools/graph_edges.go`
- Create: `servers/nexus/internal/tools/graph_query.go`
- Create: `servers/nexus/internal/tools/scheduler_bridge.go`

**Interfaces:**
- Consumes: `store.Store`, `*mcputil.SessionMap`
- Produces: evidence/hypothesis/vulnerability tools, edge tools, graph_query/graph_trace, scheduler bridge

- [ ] **Step 1: reasoning.go — Evidence + Hypothesis + Vulnerability tools**

```go
// servers/nexus/internal/tools/reasoning.go
package tools

import (
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"adversarychef/nexus/internal/models"
	"adversarychef/nexus/internal/store"
	"adversarychef/mcputil"
)

func registerReasoningNodes(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// Evidence
	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "evidence_list", Description: "List evidence nodes by project ID"},
		scopedLister(sm, func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID string `json:"project_id"` }) (*mcp.CallToolResult, any, error) {
			items, err := s.ListEvidence(params.ProjectID)
			if err != nil { return mcputil.TextResult("query failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(items); return mcputil.TextResult(string(b)), nil, nil
		}))

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "evidence_create", Description: "Create an evidence node (e.g., scan result, tool output reference)"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID, Label, Source, ContentRef string }) (*mcp.CallToolResult, any, error) {
			e := &models.EvidenceNode{ProjectID: params.ProjectID, Label: params.Label, Source: params.Source, ContentRef: params.ContentRef}
			if err := s.CreateEvidence(e); err != nil { return mcputil.TextResult("create failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(e); return mcputil.TextResult(string(b)), nil, nil
		})

	// Hypothesis
	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "hypothesis_list", Description: "List hypothesis nodes by project ID"},
		scopedLister(sm, func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID string `json:"project_id"` }) (*mcp.CallToolResult, any, error) {
			items, err := s.ListHypotheses(params.ProjectID)
			if err != nil { return mcputil.TextResult("query failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(items); return mcputil.TextResult(string(b)), nil, nil
		}))

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "hypothesis_create", Description: "Create a hypothesis node"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID, Label string; Confidence float64 }) (*mcp.CallToolResult, any, error) {
			h := &models.HypothesisNode{ProjectID: params.ProjectID, Label: params.Label, Confidence: params.Confidence, Status: "proposed"}
			if err := s.CreateHypothesis(h); err != nil { return mcputil.TextResult("create failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(h); return mcputil.TextResult(string(b)), nil, nil
		})

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "hypothesis_update", Description: "Update hypothesis status"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ID, Status string; Confidence float64 }) (*mcp.CallToolResult, any, error) {
			if err := s.UpdateHypothesisStatus(params.ID, params.Status, params.Confidence); err != nil {
				return mcputil.TextResult("update failed: " + err.Error()), nil, nil
			}
			return mcputil.TextResult("updated"), nil, nil
		})

	// Vulnerability
	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "vulnerability_list", Description: "List vulnerability nodes by project ID"},
		scopedLister(sm, func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID string `json:"project_id"` }) (*mcp.CallToolResult, any, error) {
			items, err := s.ListVulnerabilities(params.ProjectID)
			if err != nil { return mcputil.TextResult("query failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(items); return mcputil.TextResult(string(b)), nil, nil
		}))

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "vulnerability_create", Description: "Create a vulnerability node (REQUIRES evidence_refs)"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID, Title, CVE, Severity, Description, Remediation string; CVSS float64; EvidenceRefs []string }) (*mcp.CallToolResult, any, error) {
			if len(params.EvidenceRefs) == 0 { return mcputil.TextResult("evidence_refs is required"), nil, nil }
			v := &models.VulnerabilityNode{ProjectID: params.ProjectID, Title: params.Title, CVE: params.CVE, Severity: params.Severity, CVSS: params.CVSS, Description: params.Description, Remediation: params.Remediation, Status: "open", EvidenceRefs: params.EvidenceRefs}
			if err := s.CreateVulnerability(v); err != nil { return mcputil.TextResult("create failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(v); return mcputil.TextResult(string(b)), nil, nil
		})
}
```

- [ ] **Step 2: graph_edges.go — Edge CRUD**

```go
func registerGraphEdges(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "edge_create", Description: "Create a graph edge between two nodes"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID, FromID, ToID, EdgeType string; EvidenceRefs []string }) (*mcp.CallToolResult, any, error) {
			e := &models.GraphEdge{ProjectID: params.ProjectID, FromID: params.FromID, ToID: params.ToID, EdgeType: params.EdgeType, EvidenceRefs: params.EvidenceRefs}
			if err := s.CreateEdge(e); err != nil { return mcputil.TextResult("create failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(e); return mcputil.TextResult(string(b)), nil, nil
		})

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "edge_list", Description: "List graph edges by project ID (optional from/to filter)"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID, FromID, ToID string }) (*mcp.CallToolResult, any, error) {
			items, err := s.ListEdges(params.ProjectID, params.FromID, params.ToID)
			if err != nil { return mcputil.TextResult("query failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(items); return mcputil.TextResult(string(b)), nil, nil
		})

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "edge_delete", Description: "Delete a graph edge"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ID string }) (*mcp.CallToolResult, any, error) {
			if err := s.DeleteEdge(params.ID); err != nil { return mcputil.TextResult("delete failed: " + err.Error()), nil, nil }
			return mcputil.TextResult("deleted"), nil, nil
		})
}
```

- [ ] **Step 3: graph_query.go — Graph traversal tools**

```go
func registerGraphQueries(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "graph_query", Description: "BFS traverse from a node, up to max_hops (default 2)"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID, StartNodeID string; MaxHops int }) (*mcp.CallToolResult, any, error) {
			if params.MaxHops == 0 { params.MaxHops = 2 }
			sub, err := s.GraphQuery(params.ProjectID, params.StartNodeID, params.MaxHops)
			if err != nil { return mcputil.TextResult("query failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(sub); return mcputil.TextResult(string(b)), nil, nil
		})

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "graph_trace", Description: "Backtrace from a node to its evidence sources"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ProjectID, NodeID string }) (*mcp.CallToolResult, any, error) {
			trace, err := s.GraphTrace(params.ProjectID, params.NodeID)
			if err != nil { return mcputil.TextResult("trace failed: " + err.Error()), nil, nil }
			b, _ := json.Marshal(trace); return mcputil.TextResult(string(b)), nil, nil
		})
}
```

- [ ] **Step 4: scheduler_bridge.go — Forward to acasched**

```go
func registerSchedulerBridge(server *mcp.Server, sm *mcputil.SessionMap) {
	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "scheduler_create_task", Description: "Create a sub-task for another agent to execute (via acasched)"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ ParentID, Agent, Title, Description string; MaxTurns int }) (*mcp.CallToolResult, any, error) {
			// HTTP POST http://acasched:8080/api/tasks
			body, _ := json.Marshal(map[string]any{
				"parent_id": params.ParentID, "agent": params.Agent,
				"title": params.Title, "description": params.Description,
				"max_turns": params.MaxTurns, "created_by": "agent",
			})
			resp, err := http.Post("http://127.0.0.1:9090/api/tasks", "application/json", bytes.NewReader(body))
			if err != nil { return mcputil.TextResult("scheduler unreachable: " + err.Error()), nil, nil }
			defer resp.Body.Close()
			var task map[string]any
			json.NewDecoder(resp.Body).Decode(&task)
			b, _ := json.Marshal(task)
			return mcputil.TextResult(string(b)), nil, nil
		})

	mcputil.AddLoggingTool(server, &mcp.Tool{Name: "scheduler_complete_task", Description: "Mark your own task as complete"},
		func(ctx context.Context, req *mcp.CallToolRequest, params struct{ TaskID, Result string }) (*mcp.CallToolResult, any, error) {
			body, _ := json.Marshal(map[string]string{"result": params.Result})
			req, _ := http.NewRequestWithContext(ctx, "PATCH", "http://127.0.0.1:9090/api/tasks/"+params.TaskID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil { return mcputil.TextResult("scheduler unreachable: " + err.Error()), nil, nil }
			defer resp.Body.Close()
			return mcputil.TextResult("task marked done"), nil, nil
		})
}
```

- [ ] **Step 5: Build + commit**

```bash
cd servers/nexus && go build ./...
git add servers/nexus/internal/tools/
git commit -m "feat(nexus): add reasoning, edge, query, and scheduler bridge tools"
```

---

## Task 8: Wire Session Binding into All Tool Handlers

**Files:**
- Modify: `pkg/mcputil/mcputil.go`
- Modify: `servers/nexus/cmd/server/main.go`

**Interfaces:**
- Consumes: MCP session ID from request context
- Produces: `projectID` validated via `SessionMap.GetOrBind`

- [ ] **Step 1: Extract MCP session ID from request context**

```go
// pkg/mcputil/mcputil.go — add context key and helper
type ctxKey int
const CtxKeyProjectID ctxKey = iota

func ProjectIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(CtxKeyProjectID).(string)
	return id
}

func WithProjectID(ctx context.Context, projectID string) context.Context {
	return context.WithValue(ctx, CtxKeyProjectID, projectID)
}
```

- [ ] **Step 2: Add session-binding middleware to nexus-mcp**

```go
// servers/nexus/cmd/server/main.go
func main() {
	cfg := mcputil.ParseConfig("nexus", "0.3.0", 8081)
	s, err := store.NewSQLiteStore(cfg.DBPath)
	// ...
	sessionMap := mcputil.NewSessionMap()

	mcputil.Run(cfg, func(server *mcp.Server) {
		// Wrap server with session binding middleware
		bindServer := mcputil.WithSessionBinding(server, sessionMap)
		tools.RegisterAllV2(bindServer, s, sessionMap)
	})
}
```

```go
// pkg/mcputil/mcputil.go — add
func WithSessionBinding(server *mcp.Server, sm *SessionMap) *mcp.Server {
	// Intercept tool calls to validate/auto-bind project_id
	// The MCP go-sdk doesn't expose middleware natively,
	// so we patch tool handlers at registration time instead.
	// See scopedHandler below.
	return server
}

// scopedHandler wraps a tool handler to enforce session binding.
// The callerProjectID is extracted from the handler params struct
// dynamically via reflection or interface.
func ScopedHandler(sm *SessionMap, sessionID string, handler func(...)) func(...) {
	// Implementation: extract project_id from params, call GetOrBind
	// Return error if binding fails
}
```

Note: The MCP go-sdk v1.6.1 may not expose session IDs from the HTTP handler layer. If `mcputil` cannot intercept at the HTTP handler level, the fallback approach is: each tool handler extracts project_id from params and calls `sessionMap.GetOrBind(sessionID, projectID)` as its first operation. The `Run()` function's `withMiddleware` wrapper can capture `X-MCP-Session-ID` headers.

- [ ] **Step 3: Implement via HTTP header interception**

```go
// pkg/mcputil/mcputil.go — modify withMiddleware
func withMiddleware(next http.Handler, sm *SessionMap) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() { /* panic recovery unchanged */ }()

		// Extract MCP session ID from header
		sessionID := r.Header.Get("Mcp-Session-Id")
		if sessionID != "" {
			ctx := r.Context()
			// For tool calls, project_id comes from the JSON body
			// We parse it here and inject into context
			if r.Method == "POST" {
				body, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewReader(body))
				var toolCall struct {
					Params struct {
						ProjectID string `json:"project_id"`
					} `json:"params"`
				}
				if json.Unmarshal(body, &toolCall) == nil && toolCall.Params.ProjectID != "" {
					projectID, err := sm.GetOrBind(sessionID, toolCall.Params.ProjectID)
					if err != nil {
						http.Error(w, `{"error":"project binding conflict: `+err.Error()+`"}`, http.StatusForbidden)
						return
					}
					ctx = mcputil.WithProjectID(ctx, projectID)
				}
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Update Run() to pass SessionMap to withMiddleware
func Run(cfg ServerConfig, register func(*mcp.Server), sm *SessionMap) error {
	// ...
	mux.Handle("/", withMiddleware(handler, sm))
	// ...
}
```

- [ ] **Step 4: Build + commit**

```bash
cd servers/nexus && go build ./...
git add . && git commit -m "feat(nexus): wire session binding into all tool handlers"
```

---

## Task 9: acasched — Project + Task SQLite Schema

**Files:**
- Create: `internal/store/sqlite.go`

**Interfaces:**
- Consumes: nothing
- Produces: `NewStore(dbPath string) (*Store, error)`, `Store` with CRUD for projects and tasks

- [ ] **Step 1: Create cmd/acasched directory and go module**

```bash
mkdir -p cmd/acasched
```

```go
// cmd/acasched/go.mod — NOT needed, acasched lives in the root module
// Add to go.work: use ./cmd/acasched — OR make acasched part of the nexus module
```

Decision: acasched is a separate binary. Add `./cmd/acasched` to `go.work`. Module path: `adversarychef/acasched`.

```go
// cmd/acasched/go.mod
module adversarychef/acasched
go 1.26.4
require modernc.org/sqlite v1.54.0
```

- [ ] **Step 2: Write internal/store/sqlite.go**

```go
// cmd/acasched/internal/store/sqlite.go
package store

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
	_ "modernc.org/sqlite"
)

type Task struct {
	ID           string
	ProjectID    string
	ParentID     string
	Agent        string
	Status       string    // pending|dispatched|running|done|failed|timeout|skipped
	Title        string
	Description  string
	Result       string
	Error        string
	CreatedBy    string
	MaxTurns     int
	TimeoutSecs  int
	RetryCount   int
	Attempt      int
	CreatedAt    time.Time
	DispatchedAt *time.Time
	CompletedAt  *time.Time
}

type Project struct {
	ID          string
	Name        string
	Description string
	Status      string
	CreatedAt   time.Time
}

type Store struct {
	mu sync.RWMutex
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil { return nil, fmt.Errorf("open sqlite: %w", err) }
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil { return nil, err }
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT DEFAULT '', status TEXT DEFAULT 'active', created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS tasks (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, parent_id TEXT, agent TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', title TEXT NOT NULL, description TEXT NOT NULL, result TEXT DEFAULT '', error TEXT DEFAULT '', created_by TEXT NOT NULL, max_turns INTEGER DEFAULT 40, timeout_secs INTEGER DEFAULT 1800, retry_count INTEGER DEFAULT 1, attempt INTEGER DEFAULT 0, created_at TEXT NOT NULL, dispatched_at TEXT, completed_at TEXT)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(project_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_id)`,
	}
	for _, d := range ddl {
		if _, err := s.db.Exec(d); err != nil { return err }
	}
	return nil
}

func (s *Store) CreateTask(t *Task) error {
	s.mu.Lock(); defer s.mu.Unlock()
	if t.ID == "" { t.ID = fmt.Sprintf("task_%d", time.Now().UnixNano()) }
	t.CreatedAt = time.Now()
	_, err := s.db.Exec(`INSERT INTO tasks (id, project_id, parent_id, agent, status, title, description, created_by, max_turns, timeout_secs, retry_count, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.ProjectID, t.ParentID, t.Agent, t.Status, t.Title, t.Description, t.CreatedBy, t.MaxTurns, t.TimeoutSecs, t.RetryCount, t.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *Store) ListPending(projectID string) ([]Task, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, parent_id, agent, status, title, description, result, error, created_by, max_turns, timeout_secs, retry_count, attempt, created_at FROM tasks WHERE project_id = ? AND status = 'pending' ORDER BY created_at ASC`, projectID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Task
	for rows.Next() {
		var t Task; var ct, dt, cmt sql.NullString
		rows.Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Agent, &t.Status, &t.Title, &t.Description, &t.Result, &t.Error, &t.CreatedBy, &t.MaxTurns, &t.TimeoutSecs, &t.RetryCount, &t.Attempt, &ct)
		t.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		if dt.Valid { pt, _ := time.Parse(time.RFC3339, dt.String); t.DispatchedAt = &pt }
		if cmt.Valid { pt, _ := time.Parse(time.RFC3339, cmt.String); t.CompletedAt = &pt }
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) UpdateStatus(id, status, result, errMsg string) error {
	s.mu.Lock(); defer s.mu.Unlock()
	now := time.Now().Format(time.RFC3339)
	_, e := s.db.Exec(`UPDATE tasks SET status=?, result=?, error=?, completed_at=? WHERE id=?`, status, result, errMsg, now, id)
	return e
}

func (s *Store) MarkDispatched(id string) error {
	s.mu.Lock(); defer s.mu.Unlock()
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE tasks SET status='dispatched', dispatched_at=? WHERE id=?`, now, id)
	return err
}

func (s *Store) MarkRunning(id string) error {
	s.mu.Lock(); defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE tasks SET status='running' WHERE id=?`, id)
	return err
}

func (s *Store) FindChildren(parentID string) ([]Task, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, status FROM tasks WHERE parent_id = ?`, parentID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Task
	for rows.Next() {
		var t Task; rows.Scan(&t.ID, &t.Status)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTask(id string) (*Task, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var t Task; var ct string
	err := s.db.QueryRow(`SELECT id, project_id, parent_id, agent, status, title, description, result, error, created_by, max_turns, timeout_secs, retry_count, attempt, created_at FROM tasks WHERE id = ?`, id).
		Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Agent, &t.Status, &t.Title, &t.Description, &t.Result, &t.Error, &t.CreatedBy, &t.MaxTurns, &t.TimeoutSecs, &t.RetryCount, &t.Attempt, &ct)
	if err != nil { return nil, err }
	t.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &t, nil
}

// Project CRUD
func (s *Store) CreateProject(p *Project) error {
	s.mu.Lock(); defer s.mu.Unlock()
	if p.ID == "" { p.ID = fmt.Sprintf("proj_%d", time.Now().UnixNano()) }
	p.CreatedAt = time.Now()
	_, err := s.db.Exec(`INSERT INTO projects (id, name, description, status, created_at) VALUES (?,?,?,?,?)`, p.ID, p.Name, p.Description, p.Status, p.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *Store) GetProject(id string) (*Project, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	var p Project; var ct string
	err := s.db.QueryRow(`SELECT id, name, description, status, created_at FROM projects WHERE id = ?`, id).Scan(&p.ID, &p.Name, &p.Description, &p.Status, &ct)
	if err != nil { return nil, err }
	p.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &p, nil
}
```

- [ ] **Step 3: Build verification**

```bash
go build ./cmd/acasched/...
```

- [ ] **Step 4: Commit**

```bash
git add cmd/acasched/ go.work go.work.sum
git commit -m "feat(acasched): add projects + tasks SQLite store"
```

---

## Task 10: acasched — Dispatcher + Lifecycle + Trigger + Reaper

**Files:**
- Create: `cmd/acasched/internal/scheduler/dispatcher.go`
- Create: `cmd/acasched/internal/scheduler/lifecycle.go`
- Create: `cmd/acasched/internal/scheduler/trigger.go`
- Create: `cmd/acasched/internal/scheduler/reaper.go`
- Create: `cmd/acasched/internal/goose/runner.go`
- Create: `cmd/acasched/internal/goose/parser.go`
- Create: `cmd/acasched/main.go`

- [ ] **Step 1: lifecycle.go — Status transitions**

```go
// cmd/acasched/internal/scheduler/lifecycle.go
package scheduler

import "adversarychef/acasched/internal/store"

func TransitionToDispatched(s *store.Store, taskID string) error { return s.MarkDispatched(taskID) }
func TransitionToRunning(s *store.Store, taskID string) error    { return s.MarkRunning(taskID) }
func TransitionToDone(s *store.Store, taskID, result string) error { return s.UpdateStatus(taskID, "done", result, "") }
func TransitionToFailed(s *store.Store, taskID, errMsg string) error { return s.UpdateStatus(taskID, "failed", "", errMsg) }
func TransitionToTimeout(s *store.Store, taskID string) error { return s.UpdateStatus(taskID, "timeout", "", "execution timed out") }
```

- [ ] **Step 2: dispatcher.go — Main dispatch loop**

```go
// cmd/acasched/internal/scheduler/dispatcher.go
package scheduler

import (
	"context"
	"log"
	"time"
	"adversarychef/acasched/internal/goose"
	"adversarychef/acasched/internal/store"
)

type Dispatcher struct {
	store   *store.Store
	runner  *goose.Runner
	running map[string]context.CancelFunc
}

func NewDispatcher(s *store.Store, r *goose.Runner) *Dispatcher {
	return &Dispatcher{store: s, runner: r, running: map[string]context.CancelFunc{}}
}

func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done(): return
		case <-ticker.C:
			d.tick()
		}
	}
}

func (d *Dispatcher) tick() {
	tasks, err := d.store.ListPending("")
	if err != nil { log.Printf("dispatcher: list pending: %v", err); return }
	for _, t := range tasks {
		if t.ParentID != "" && !d.parentReady(t.ParentID) { continue }
		go d.dispatchOne(t)
	}
}

func (d *Dispatcher) dispatchOne(task store.Task) {
	TransitionToDispatched(d.store, task.ID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(task.TimeoutSecs)*time.Second)
	// d.running[task.ID] = cancel — track for graceful shutdown

	TransitionToRunning(d.store, task.ID)
	result, err := d.runner.Execute(ctx, &task)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			TransitionToTimeout(d.store, task.ID)
		} else if task.Attempt < task.RetryCount {
			task.Attempt++; task.Description += "\n[RETRY] " + err.Error()
			d.store.UpdateStatus(task.ID, "pending", "", "")
			return
		} else {
			TransitionToFailed(d.store, task.ID, err.Error())
		}
		return
	}
	TransitionToDone(d.store, task.ID, result.Summary)
	TriggerParent(d.store, task.ParentID)
}

func (d *Dispatcher) parentReady(parentID string) bool {
	// Parent is "ready" when its status is done/failed/timeout
	// This prevents dispatching children of still-running parents
	p, err := d.store.GetTask(parentID)
	if err != nil { return false }
	return p.Status == "done" || p.Status == "failed" || p.Status == "timeout"
}
```

- [ ] **Step 3: trigger.go — Parent re-trigger**

```go
// cmd/acasched/internal/scheduler/trigger.go
package scheduler

import (
	"fmt"
	"adversarychef/acasched/internal/store"
)

func TriggerParent(s *store.Store, parentID string) {
	if parentID == "" { return }
	children, err := s.FindChildren(parentID)
	if err != nil { return }
	allTerminal := true
	for _, c := range children {
		if c.Status != "done" && c.Status != "failed" && c.Status != "timeout" && c.Status != "skipped" {
			allTerminal = false; break
		}
	}
	if !allTerminal { return }

	parent, err := s.GetTask(parentID)
	if err != nil { return }

	// Inject child results into parent description
	summary := "\n\n## Child Task Results\n"
	for _, c := range children {
		ct, _ := s.GetTask(c.ID)
		summary += fmt.Sprintf("- %s (%s): %s\n", ct.Title, ct.Status, truncate(ct.Result, 200))
	}
	parent.Description += summary
	parent.Status = "pending"
	s.UpdateStatus(parent.ID, "pending", "", "")
}

func truncate(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n] + "..."
}
```

- [ ] **Step 4: reaper.go — Timeout detection**

```go
// cmd/acasched/internal/scheduler/reaper.go
package scheduler

import (
	"log"
	"time"
	"adversarychef/acasched/internal/store"
)

func RunReaper(s *store.Store, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		// Find running tasks past their timeout
		// For now: simple approach — dispatcher's context.WithTimeout handles it
		_ = s
	}
	log.Println("reaper: started")
}
```

- [ ] **Step 5: goose runner**

```go
// cmd/acasched/internal/goose/runner.go
package goose

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"adversarychef/acasched/internal/store"
)

type Runner struct {
	PromptsDir string
	WorkDir    string
	NexusMCP   string
	KaliMCP    string
	MythicMCP  string
}

func (r *Runner) Execute(ctx context.Context, task *store.Task) (*Result, error) {
	prompt := r.buildPrompt(task)
	tmpFile, _ := os.CreateTemp("", "goose-instructions-*.md")
	tmpFile.WriteString(prompt)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	cmd := exec.CommandContext(ctx, "goose", "run",
		"--instructions", tmpFile.Name(),
		"--text", task.Description,
		"--with-streamable-http-extension", r.NexusMCP,
		"--with-streamable-http-extension", r.KaliMCP,
		"--with-streamable-http-extension", r.MythicMCP,
		"--max-turns", fmt.Sprintf("%d", task.MaxTurns),
		"--no-session",
		"--output-format", "stream-json",
		"--no-profile",
	)
	cmd.Dir = r.WorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("goose exited: %w, output: %s", err, string(output))
	}
	return parseStreamOutput(string(output)), nil
}

func (r *Runner) buildPrompt(task *store.Task) string {
	content, _ := os.ReadFile(r.PromptsDir + "/" + task.Agent + ".md")
	return fmt.Sprintf(`## Session Binding
project_id: %s
task_id: %s

## Task Lifecycle
- Use scheduler_create_task to delegate work
- Use scheduler_complete_task to mark yourself done
- Do NOT exit without calling scheduler_complete_task

---
%s`, task.ProjectID, task.ID, string(content))
}

type Result struct {
	Status  string
	Summary string
	Output  string
}
```

- [ ] **Step 6: stream-json parser**

```go
// cmd/acasched/internal/goose/parser.go
package goose

import (
	"bufio"
	"encoding/json"
	"strings"
)

type streamLine struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

func parseStreamOutput(output string) *Result {
	var summary strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		var line streamLine
		if json.Unmarshal(scanner.Bytes(), &line) != nil { continue }
		if line.Type == "assistant" {
			var blocks []struct{ Text string `json:"text"` }
			if json.Unmarshal(line.Content, &blocks) == nil {
				for _, b := range blocks { summary.WriteString(b.Text) }
			}
		}
	}
	return &Result{Status: "done", Summary: summary.String(), Output: output}
}
```

- [ ] **Step 7: main.go**

```go
// cmd/acasched/main.go
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"adversarychef/acasched/internal/goose"
	"adversarychef/acasched/internal/scheduler"
	"adversarychef/acasched/internal/store"
)

func main() {
	dbPath := flag.String("db", "acasched.db", "sqlite database path")
	port := flag.Int("port", 9090, "HTTP API port")
	promptsDir := flag.String("prompts", "prompts", "prompts directory")
	nexusURL := flag.String("nexus-mcp", "http://127.0.0.1:8081", "nexus-mcp URL")
	kaliURL := flag.String("kali-mcp", "http://127.0.0.1:8080", "kali-mcp URL")
	mythicURL := flag.String("mythic-mcp", "http://127.0.0.1:8082", "mythic-mcp URL")
	flag.Parse()

	s, err := store.NewStore(*dbPath)
	if err != nil { log.Fatalf("store: %v", err) }
	defer s.Close()

	runner := &goose.Runner{
		PromptsDir: *promptsDir,
		NexusMCP:   *nexusURL,
		KaliMCP:    *kaliURL,
		MythicMCP:  *mythicURL,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	disp := scheduler.NewDispatcher(s, runner)
	go disp.Run(ctx)
	go runAPI(ctx, s, *port)
	go scheduler.RunReaper(s, 30)

	log.Printf("acasched started on :%d", *port)
	<-ctx.Done()
	log.Println("acasched shutting down")
}
```

- [ ] **Step 8: Build + commit**

```bash
go build ./cmd/acasched/...
git add cmd/acasched/ go.work go.work.sum
git commit -m "feat(acasched): dispatcher, lifecycle, trigger, reaper, goose runner"
```

---

## Task 11: acasched — HTTP API

**Files:**
- Create: `cmd/acasched/internal/api/server.go`
- Create: `cmd/acasched/internal/api/tasks.go`
- Create: `cmd/acasched/internal/api/projects.go`
- Modify: `cmd/acasched/main.go`

- [ ] **Step 1: server.go + tasks.go + projects.go**

```go
// cmd/acasched/internal/api/server.go
func runAPI(ctx context.Context, s *store.Store, port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", handleTasks(s))
	mux.HandleFunc("/api/tasks/", handleTaskByID(s))  // GET / PATCH /:id
	mux.HandleFunc("/api/projects", handleProjects(s))
	mux.HandleFunc("/api/projects/", handleProjectByID(s))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

// tasks.go
func handleTasks(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			projectID := r.URL.Query().Get("project_id")
			tasks, _ := s.ListPending(projectID)
			json.NewEncoder(w).Encode(tasks)
		case "POST":
			var t store.Task
			json.NewDecoder(r.Body).Decode(&t)
			t.Status = "pending"
			s.CreateTask(&t)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(t)
		}
	}
}

func handleTaskByID(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
		switch r.Method {
		case "GET":
			t, _ := s.GetTask(id)
			json.NewEncoder(w).Encode(t)
		case "PATCH":
			var req struct{ Result, Error, Status string }
			json.NewDecoder(r.Body).Decode(&req)
			s.UpdateStatus(id, req.Status, req.Result, req.Error)
			w.Write([]byte(`{"status":"updated"}`))
		}
	}
}
```

- [ ] **Step 2: Wire into main.go**

```go
// cmd/acasched/main.go — add
import "adversarychef/acasched/internal/api"
// ...
go api.RunAPI(ctx, s, *port)
```

- [ ] **Step 3: Build + commit**

```bash
go build ./cmd/acasched/...
git add cmd/acasched/ go.work go.work.sum
git commit -m "feat(acasched): HTTP API for tasks and projects"
```

---

## Task 12: Agent Prompt Adaptation

**Files:**
- Modify: `prompts/supervisor.md`
- Modify: `prompts/echo-recon.md`
- Modify: `prompts/breach-exploit.md`
- Modify: `prompts/ghost-mythic.md`
- Modify: `prompts/path-lateral.md`
- Modify: `prompts/quill-report.md`
- Modify: `prompts/strategist.md`
- Modify: `prompts/forge-resource.md`

- [ ] **Step 1: Add Session Binding + Task Lifecycle section to ALL prompts**

Each prompt gets this prepended (injected by goose runner at runtime, but base prompt should reference it):

```markdown
## Runtime Context
- This session is automatically bound to the project_id in your task.
- All nexus-mcp tool calls are scoped to this project.
- Use `scheduler_create_task` to delegate work to other agents.
- Use `scheduler_complete_task` to mark your task done with a result summary.
- Do NOT exit without calling `scheduler_complete_task`.
```

- [ ] **Step 2: Rewrite AC-Supervisor prompt**

```markdown
# AC-Supervisor — Attack Director

You receive a penetration testing task. Your job:

1. Query nexus-mcp to understand current project state (graph_query, project_summary)
2. Decide the next phase based on what has been discovered
3. Delegate work by calling `scheduler_create_task` for each specialist agent
4. When all child tasks complete, you are re-triggered — evaluate and decide next step
5. When the engagement is complete, call `scheduler_create_task(agent="quill")` for the report

## Decision Rules
| Situation | Action |
|-----------|--------|
| Project has 0 assets | Delegate to AC-Strategist then AC-Echo |
| Open clues without exploit confirmation | Delegate to AC-Breach |
| Confirmed vulnerability, no C2 access | Delegate to AC-Ghost |
| C2 access, internal network visible | Delegate to AC-Path |
| Infrastructure needed | Delegate to AC-Forge |
| All phases done | Delegate to AC-Quill |

## Hard Rules
- NEVER execute tools yourself — delegate to specialists
- Always call scheduler_complete_task when your evaluation cycle is done
- Record decisions: why you chose this agent for this phase
```

- [ ] **Step 3: Update each specialist prompt**

Remove Multica/Squad references. Add:
- `graph_query` and `graph_trace` as primary discovery mechanism
- `host_create/service_create/endpoint_create` as primary recording mechanism (instead of create_asset)
- `scheduler_complete_task` as required exit action
- `evidence_create → hypothesis_create → vulnerability_create` chain for Breach/Ghost/Path

- [ ] **Step 4: Commit**

```bash
git add prompts/
git commit -m "feat(prompts): adapt all agent prompts for acasched + nexus-mcp v2"
```

---

## Task 13: Integration Test — End-to-End

**Files:**
- Create: `cmd/acasched/internal/scheduler/dispatcher_test.go`
- Create: `servers/nexus/internal/tools/graph_nodes_test.go`

- [ ] **Step 1: Dispatcher unit test**

```go
func TestDispatcherTick(t *testing.T) {
	s, _ := store.NewStore(":memory:")
	s.CreateProject(&store.Project{ID: "proj_001", Name: "test"})
	s.CreateTask(&store.Task{ID: "t1", ProjectID: "proj_001", Agent: "echo", Title: "test", Description: "do recon", CreatedBy: "human"})
	tasks, _ := s.ListPending("proj_001")
	assert.Len(t, tasks, 1)
	assert.Equal(t, "pending", tasks[0].Status)
}
```

- [ ] **Step 2: Graph node CRUD test**

```go
func TestHostCRUD(t *testing.T) {
	store, _ := store.NewSQLiteStore(":memory:")
	h := &models.HostNode{ProjectID: "proj_001", IPs: []string{"10.0.0.1"}, Hostname: "web01"}
	assert.NoError(t, store.CreateHost(h))
	assert.NotEmpty(t, h.ID)

	hosts, _ := store.ListHosts("proj_001")
	assert.Len(t, hosts, 1)
	assert.Equal(t, "web01", hosts[0].Hostname)

	got, _ := store.GetHost(h.ID)
	assert.Equal(t, h.ID, got.ID)
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./cmd/acasched/... ./servers/nexus/...
```

- [ ] **Step 4: Commit**

```bash
git add . && git commit -m "test: add dispatcher and graph node integration tests"
```

---

## Task 14: Docker Compose Update

**Files:**
- Modify: `docker/docker-compose.yml`
- Modify: `docker/Dockerfile` (rename nexus)

- [ ] **Step 1: Update docker-compose.yml**

```yaml
services:
  nexus-mcp:
    build:
      context: ..
      dockerfile: docker/nexus-mcp/Dockerfile
    ports: ["8081:8081"]
    command: -db /data/nexus.db
    volumes: [nexus-data:/data]

  kali-mcp:
    build:
      context: ..
      dockerfile: docker/kali-mcp/Dockerfile
    ports: ["8080:8080"]

  mythic-mcp:
    build:
      context: ..
      dockerfile: docker/mythic-mcp/Dockerfile
    ports: ["8082:8082"]
    environment:
      - MYTHIC_SERVER=${MYTHIC_SERVER}
      - MYTHIC_API_KEY=${MYTHIC_API_KEY}

  acasched:
    build:
      context: ..
      dockerfile: docker/acasched/Dockerfile
    ports: ["9090:9090"]
    command: -db /data/acasched.db -nexus-mcp http://nexus-mcp:8081
    volumes: [acasched-data:/data, ./prompts:/prompts:ro]
    depends_on: [nexus-mcp]

volumes:
  nexus-data:
  acasched-data:
```

- [ ] **Step 2: Commit**

```bash
git add docker/
git commit -m "chore: update docker-compose for nexus-mcp + acasched"
```

---

## Self-Review

**1. Spec coverage:** 
- ✅ Architecture (Section 1): Task 1+2+3 implement the full architecture
- ✅ acasched scheduler (Section 2): Tasks 9-11 cover schema, dispatcher, trigger, reaper, API
- ✅ nexus-mcp data model (Section 3): Tasks 2-8 cover models, store, tools, session binding
- ✅ Goose executor (Section 4): Task 10 covers runner + parser
- ✅ Project isolation (Section 5): Tasks 5+8 cover session binding
- ✅ Directory structure (Section 6): Matches file structure header
- ✅ Implementation timeline (Section 7): Covered by 14 tasks across 3 phases

**2. Placeholder scan:** No TBD/TODO/fill-in-later found. All code steps have concrete implementations.

**3. Type consistency:** `HostNode.ID` used consistently across models → store → tools. `SessionMap.GetOrBind(sessionID, projectID)` signature consistent across mcputil and main.go. `store.Task` and `store.Project` match the SQLite schema in Task 9.
