package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"adversarychef/nexus/internal/models"
)

// SQLiteStore is a SQLite-backed persistent store implementation.
type SQLiteStore struct {
	mu sync.RWMutex
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			status TEXT DEFAULT 'active',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS assets (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			ips TEXT DEFAULT '[]',
			domains TEXT DEFAULT '[]',
			tech_stack TEXT DEFAULT '[]',
			scope TEXT DEFAULT '',
			description TEXT DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS clues (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT DEFAULT '',
			type TEXT DEFAULT '',
			status TEXT DEFAULT 'open',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS credentials (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			asset_id TEXT DEFAULT '',
			credential_type TEXT NOT NULL,
			label TEXT NOT NULL,
			value TEXT NOT NULL,
			expires_at TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS worklogs (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT DEFAULT '',
			created_at TEXT NOT NULL
		)`,
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
	}
	for _, d := range ddl {
		if _, err := s.db.Exec(d); err != nil {
			return fmt.Errorf("exec ddl: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Project CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListProjects() ([]models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, name, description, status, created_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Project
	for rows.Next() {
		var p models.Project
		var ct string
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &ct); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetProject(id string) (*models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var p models.Project
	var ct string
	err := s.db.QueryRow(`SELECT id, name, description, status, created_at FROM projects WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Status, &ct)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &p, nil
}

func (s *SQLiteStore) CreateProject(p *models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = genID("proj")
	}
	p.CreatedAt = time.Now()
	_, err := s.db.Exec(`INSERT INTO projects (id, name, description, status, created_at) VALUES (?,?,?,?,?)`,
		p.ID, p.Name, p.Description, p.Status, p.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateProject(p *models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE projects SET name=?, description=?, status=? WHERE id=?`,
		p.Name, p.Description, p.Status, p.ID)
	return err
}

func (s *SQLiteStore) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------------------
// Asset CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListAssets(projectID string) ([]models.Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, name, ips, domains, tech_stack, scope, description, created_at FROM assets WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Asset
	for rows.Next() {
		var a models.Asset
		var ips, domains, ts, ct string
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Name, &ips, &domains, &ts, &a.Scope, &a.Description, &ct); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(ips), &a.IPs)
		json.Unmarshal([]byte(domains), &a.Domains)
		json.Unmarshal([]byte(ts), &a.TechStack)
		a.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetAsset(id string) (*models.Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var a models.Asset
	var ips, domains, ts, ct string
	err := s.db.QueryRow(`SELECT id, project_id, name, ips, domains, tech_stack, scope, description, created_at FROM assets WHERE id = ?`, id).
		Scan(&a.ID, &a.ProjectID, &a.Name, &ips, &domains, &ts, &a.Scope, &a.Description, &ct)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("asset not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(ips), &a.IPs)
	json.Unmarshal([]byte(domains), &a.Domains)
	json.Unmarshal([]byte(ts), &a.TechStack)
	a.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &a, nil
}

func (s *SQLiteStore) CreateAsset(a *models.Asset) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.ID == "" {
		a.ID = genID("asset")
	}
	a.CreatedAt = time.Now()
	ips, _ := json.Marshal(emptySlice(a.IPs))
	domains, _ := json.Marshal(emptySlice(a.Domains))
	ts, _ := json.Marshal(emptySlice(a.TechStack))
	_, err := s.db.Exec(`INSERT INTO assets (id, project_id, name, ips, domains, tech_stack, scope, description, created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		a.ID, a.ProjectID, a.Name, string(ips), string(domains), string(ts), a.Scope, a.Description, a.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateAsset(a *models.Asset) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ips, _ := json.Marshal(emptySlice(a.IPs))
	domains, _ := json.Marshal(emptySlice(a.Domains))
	ts, _ := json.Marshal(emptySlice(a.TechStack))
	_, err := s.db.Exec(`UPDATE assets SET name=?, ips=?, domains=?, tech_stack=?, scope=?, description=? WHERE id=?`,
		a.Name, string(ips), string(domains), string(ts), a.Scope, a.Description, a.ID)
	return err
}

func (s *SQLiteStore) DeleteAsset(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM assets WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------------------
// Clue CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListClues(projectID string) ([]models.Clue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, title, content, type, status, created_at FROM clues WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Clue
	for rows.Next() {
		var c models.Clue
		var ct string
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Title, &c.Content, &c.Type, &c.Status, &ct); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetClue(id string) (*models.Clue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var c models.Clue
	var ct string
	err := s.db.QueryRow(`SELECT id, project_id, title, content, type, status, created_at FROM clues WHERE id = ?`, id).
		Scan(&c.ID, &c.ProjectID, &c.Title, &c.Content, &c.Type, &c.Status, &ct)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("clue not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &c, nil
}

func (s *SQLiteStore) CreateClue(c *models.Clue) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = genID("clue")
	}
	c.CreatedAt = time.Now()
	_, err := s.db.Exec(`INSERT INTO clues (id, project_id, title, content, type, status, created_at) VALUES (?,?,?,?,?,?,?)`,
		c.ID, c.ProjectID, c.Title, c.Content, c.Type, c.Status, c.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateClue(c *models.Clue) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE clues SET title=?, content=?, type=?, status=? WHERE id=?`,
		c.Title, c.Content, c.Type, c.Status, c.ID)
	return err
}

func (s *SQLiteStore) DeleteClue(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM clues WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------------------
// Credential CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListCredentials(projectID string) ([]models.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, asset_id, credential_type, label, value, expires_at, notes, created_at FROM credentials WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Credential
	for rows.Next() {
		var c models.Credential
		var ct string
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.AssetID, &c.CredentialType, &c.Label, &c.Value, &c.ExpiresAt, &c.Notes, &ct); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetCredential(id string) (*models.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var c models.Credential
	var ct string
	err := s.db.QueryRow(`SELECT id, project_id, asset_id, credential_type, label, value, expires_at, notes, created_at FROM credentials WHERE id = ?`, id).
		Scan(&c.ID, &c.ProjectID, &c.AssetID, &c.CredentialType, &c.Label, &c.Value, &c.ExpiresAt, &c.Notes, &ct)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("credential not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &c, nil
}

func (s *SQLiteStore) CreateCredential(c *models.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = genID("cred")
	}
	c.CreatedAt = time.Now()
	_, err := s.db.Exec(`INSERT INTO credentials (id, project_id, asset_id, credential_type, label, value, expires_at, notes, created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		c.ID, c.ProjectID, c.AssetID, c.CredentialType, c.Label, c.Value, c.ExpiresAt, c.Notes, c.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateCredential(c *models.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE credentials SET asset_id=?, label=?, value=?, expires_at=?, notes=? WHERE id=?`,
		c.AssetID, c.Label, c.Value, c.ExpiresAt, c.Notes, c.ID)
	return err
}

func (s *SQLiteStore) DeleteCredential(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM credentials WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------------------
// WorkLog CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListWorkLogs(projectID string) ([]models.WorkLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, title, content, created_at FROM worklogs WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.WorkLog
	for rows.Next() {
		var w models.WorkLog
		var ct string
		if err := rows.Scan(&w.ID, &w.ProjectID, &w.Title, &w.Content, &ct); err != nil {
			return nil, err
		}
		w.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetWorkLog(id string) (*models.WorkLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var w models.WorkLog
	var ct string
	err := s.db.QueryRow(`SELECT id, project_id, title, content, created_at FROM worklogs WHERE id = ?`, id).
		Scan(&w.ID, &w.ProjectID, &w.Title, &w.Content, &ct)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("worklog not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	w.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &w, nil
}

func (s *SQLiteStore) CreateWorkLog(w *models.WorkLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w.ID == "" {
		w.ID = genID("wl")
	}
	w.CreatedAt = time.Now()
	_, err := s.db.Exec(`INSERT INTO worklogs (id, project_id, title, content, created_at) VALUES (?,?,?,?,?)`,
		w.ID, w.ProjectID, w.Title, w.Content, w.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateWorkLog(w *models.WorkLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE worklogs SET title=?, content=? WHERE id=?`,
		w.Title, w.Content, w.ID)
	return err
}

func (s *SQLiteStore) DeleteWorkLog(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM worklogs WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------------------
// Host CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListHosts(projectID string) ([]models.HostNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, ips, hostname, os, evidence_refs, created_at FROM host_nodes WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.HostNode
	for rows.Next() {
		var h models.HostNode
		var ips, refs, ct string
		if err := rows.Scan(&h.ID, &h.ProjectID, &ips, &h.Hostname, &h.OS, &refs, &ct); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(ips), &h.IPs)
		json.Unmarshal([]byte(refs), &h.EvidenceRefs)
		h.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetHost(id string) (*models.HostNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var h models.HostNode
	var ips, refs, ct string
	err := s.db.QueryRow(`SELECT id, project_id, ips, hostname, os, evidence_refs, created_at FROM host_nodes WHERE id = ?`, id).
		Scan(&h.ID, &h.ProjectID, &ips, &h.Hostname, &h.OS, &refs, &ct)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("host not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(ips), &h.IPs)
	json.Unmarshal([]byte(refs), &h.EvidenceRefs)
	h.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &h, nil
}

func (s *SQLiteStore) CreateHost(h *models.HostNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h.ID == "" {
		h.ID = genID("host")
	}
	h.CreatedAt = time.Now()
	ips, _ := json.Marshal(emptySlice(h.IPs))
	refs, _ := json.Marshal(emptySlice(h.EvidenceRefs))
	_, err := s.db.Exec(`INSERT INTO host_nodes (id, project_id, ips, hostname, os, evidence_refs, created_at) VALUES (?,?,?,?,?,?,?)`,
		h.ID, h.ProjectID, string(ips), h.Hostname, h.OS, string(refs), h.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateHost(h *models.HostNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ips, _ := json.Marshal(emptySlice(h.IPs))
	refs, _ := json.Marshal(emptySlice(h.EvidenceRefs))
	_, err := s.db.Exec(`UPDATE host_nodes SET ips=?, hostname=?, os=?, evidence_refs=? WHERE id=?`,
		string(ips), h.Hostname, h.OS, string(refs), h.ID)
	return err
}

func (s *SQLiteStore) DeleteHost(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM host_nodes WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------------------
// Service CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListServices(projectID string) ([]models.ServiceNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, host_id, port, protocol, name, version, banner, evidence_refs, created_at FROM service_nodes WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ServiceNode
	for rows.Next() {
		var sv models.ServiceNode
		var refs, ct string
		if err := rows.Scan(&sv.ID, &sv.ProjectID, &sv.HostID, &sv.Port, &sv.Protocol, &sv.Name, &sv.Version, &sv.Banner, &refs, &ct); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(refs), &sv.EvidenceRefs)
		sv.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, sv)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetService(id string) (*models.ServiceNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sv models.ServiceNode
	var refs, ct string
	err := s.db.QueryRow(`SELECT id, project_id, host_id, port, protocol, name, version, banner, evidence_refs, created_at FROM service_nodes WHERE id = ?`, id).
		Scan(&sv.ID, &sv.ProjectID, &sv.HostID, &sv.Port, &sv.Protocol, &sv.Name, &sv.Version, &sv.Banner, &refs, &ct)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("service not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(refs), &sv.EvidenceRefs)
	sv.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &sv, nil
}

func (s *SQLiteStore) CreateService(sv *models.ServiceNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sv.ID == "" {
		sv.ID = genID("svc")
	}
	sv.CreatedAt = time.Now()
	refs, _ := json.Marshal(emptySlice(sv.EvidenceRefs))
	_, err := s.db.Exec(`INSERT INTO service_nodes (id, project_id, host_id, port, protocol, name, version, banner, evidence_refs, created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		sv.ID, sv.ProjectID, sv.HostID, sv.Port, sv.Protocol, sv.Name, sv.Version, sv.Banner, string(refs), sv.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateService(sv *models.ServiceNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	refs, _ := json.Marshal(emptySlice(sv.EvidenceRefs))
	_, err := s.db.Exec(`UPDATE service_nodes SET host_id=?, port=?, protocol=?, name=?, version=?, banner=?, evidence_refs=? WHERE id=?`,
		sv.HostID, sv.Port, sv.Protocol, sv.Name, sv.Version, sv.Banner, string(refs), sv.ID)
	return err
}

func (s *SQLiteStore) DeleteService(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM service_nodes WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------------------
// Endpoint CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListEndpoints(projectID string) ([]models.EndpointNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, service_id, url, method, parameters, status, discovered_by, tested_by, evidence_refs, created_at FROM endpoint_nodes WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.EndpointNode
	for rows.Next() {
		var ep models.EndpointNode
		var params, refs, ct string
		if err := rows.Scan(&ep.ID, &ep.ProjectID, &ep.ServiceID, &ep.URL, &ep.Method, &params, &ep.Status, &ep.DiscoveredBy, &ep.TestedBy, &refs, &ct); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(params), &ep.Parameters)
		json.Unmarshal([]byte(refs), &ep.EvidenceRefs)
		ep.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, ep)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetEndpoint(id string) (*models.EndpointNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var ep models.EndpointNode
	var params, refs, ct string
	err := s.db.QueryRow(`SELECT id, project_id, service_id, url, method, parameters, status, discovered_by, tested_by, evidence_refs, created_at FROM endpoint_nodes WHERE id = ?`, id).
		Scan(&ep.ID, &ep.ProjectID, &ep.ServiceID, &ep.URL, &ep.Method, &params, &ep.Status, &ep.DiscoveredBy, &ep.TestedBy, &refs, &ct)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("endpoint not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(params), &ep.Parameters)
	json.Unmarshal([]byte(refs), &ep.EvidenceRefs)
	ep.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &ep, nil
}

func (s *SQLiteStore) CreateEndpoint(ep *models.EndpointNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ep.ID == "" {
		ep.ID = genID("ep")
	}
	ep.CreatedAt = time.Now()
	params, _ := json.Marshal(emptySlice(ep.Parameters))
	refs, _ := json.Marshal(emptySlice(ep.EvidenceRefs))
	_, err := s.db.Exec(`INSERT INTO endpoint_nodes (id, project_id, service_id, url, method, parameters, status, discovered_by, tested_by, evidence_refs, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		ep.ID, ep.ProjectID, ep.ServiceID, ep.URL, ep.Method, string(params), ep.Status, ep.DiscoveredBy, ep.TestedBy, string(refs), ep.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateEndpointStatus(id, status, testedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE endpoint_nodes SET status=?, tested_by=? WHERE id=?`, status, testedBy, id)
	return err
}

func (s *SQLiteStore) GetUntestedEndpoints(projectID string) ([]models.EndpointNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, service_id, url, method, parameters, status, discovered_by, tested_by, evidence_refs, created_at FROM endpoint_nodes WHERE project_id = ? AND status = 'discovered' ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.EndpointNode
	for rows.Next() {
		var ep models.EndpointNode
		var params, refs, ct string
		if err := rows.Scan(&ep.ID, &ep.ProjectID, &ep.ServiceID, &ep.URL, &ep.Method, &params, &ep.Status, &ep.DiscoveredBy, &ep.TestedBy, &refs, &ct); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(params), &ep.Parameters)
		json.Unmarshal([]byte(refs), &ep.EvidenceRefs)
		ep.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, ep)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Session CRUD
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListSessions(projectID string) ([]models.SessionNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, asset_id, created_by, session_type, url, cookies, token_value, metadata, evidence_refs, created_at, expires_at FROM session_nodes WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SessionNode
	for rows.Next() {
		var sn models.SessionNode
		var refs, ct string
		if err := rows.Scan(&sn.ID, &sn.ProjectID, &sn.AssetID, &sn.CreatedBy, &sn.SessionType, &sn.URL, &sn.Cookies, &sn.TokenValue, &sn.Metadata, &refs, &ct, &sn.ExpiresAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(refs), &sn.EvidenceRefs)
		sn.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, sn)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetSession(id string) (*models.SessionNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sn models.SessionNode
	var refs, ct string
	err := s.db.QueryRow(`SELECT id, project_id, asset_id, created_by, session_type, url, cookies, token_value, metadata, evidence_refs, created_at, expires_at FROM session_nodes WHERE id = ?`, id).
		Scan(&sn.ID, &sn.ProjectID, &sn.AssetID, &sn.CreatedBy, &sn.SessionType, &sn.URL, &sn.Cookies, &sn.TokenValue, &sn.Metadata, &refs, &ct, &sn.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(refs), &sn.EvidenceRefs)
	sn.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &sn, nil
}

func (s *SQLiteStore) CreateSession(sn *models.SessionNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sn.ID == "" {
		sn.ID = genID("sess")
	}
	sn.CreatedAt = time.Now()
	refs, _ := json.Marshal(emptySlice(sn.EvidenceRefs))
	_, err := s.db.Exec(`INSERT INTO session_nodes (id, project_id, asset_id, created_by, session_type, url, cookies, token_value, metadata, evidence_refs, created_at, expires_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		sn.ID, sn.ProjectID, sn.AssetID, sn.CreatedBy, sn.SessionType, sn.URL, sn.Cookies, sn.TokenValue, sn.Metadata, string(refs), sn.CreatedAt.Format(time.RFC3339), sn.ExpiresAt)
	return err
}

func (s *SQLiteStore) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM session_nodes WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------------------
// Evidence CRUD (ReasoningStore)
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListEvidence(projectID string) ([]models.EvidenceNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, label, source, content_ref, evidence_refs, created_at FROM evidence_nodes WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.EvidenceNode
	for rows.Next() {
		var ev models.EvidenceNode
		var refs, ct string
		if err := rows.Scan(&ev.ID, &ev.ProjectID, &ev.Label, &ev.Source, &ev.ContentRef, &refs, &ct); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(refs), &ev.EvidenceRefs)
		ev.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CreateEvidence(ev *models.EvidenceNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev.ID == "" {
		ev.ID = genID("evid")
	}
	ev.CreatedAt = time.Now()
	refs, _ := json.Marshal(emptySlice(ev.EvidenceRefs))
	_, err := s.db.Exec(`INSERT INTO evidence_nodes (id, project_id, label, source, content_ref, evidence_refs, created_at) VALUES (?,?,?,?,?,?,?)`,
		ev.ID, ev.ProjectID, ev.Label, ev.Source, ev.ContentRef, string(refs), ev.CreatedAt.Format(time.RFC3339))
	return err
}

// ---------------------------------------------------------------------------
// Hypothesis CRUD (ReasoningStore)
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListHypotheses(projectID string) ([]models.HypothesisNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, label, confidence, status, evidence_refs, created_at FROM hypothesis_nodes WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.HypothesisNode
	for rows.Next() {
		var hy models.HypothesisNode
		var refs, ct string
		if err := rows.Scan(&hy.ID, &hy.ProjectID, &hy.Label, &hy.Confidence, &hy.Status, &refs, &ct); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(refs), &hy.EvidenceRefs)
		hy.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, hy)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CreateHypothesis(hy *models.HypothesisNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hy.ID == "" {
		hy.ID = genID("hyp")
	}
	hy.CreatedAt = time.Now()
	refs, _ := json.Marshal(emptySlice(hy.EvidenceRefs))
	_, err := s.db.Exec(`INSERT INTO hypothesis_nodes (id, project_id, label, confidence, status, evidence_refs, created_at) VALUES (?,?,?,?,?,?,?)`,
		hy.ID, hy.ProjectID, hy.Label, hy.Confidence, hy.Status, string(refs), hy.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateHypothesisStatus(id, status string, confidence float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE hypothesis_nodes SET status=?, confidence=? WHERE id=?`, status, confidence, id)
	return err
}

// ---------------------------------------------------------------------------
// Vulnerability CRUD (ReasoningStore)
// ---------------------------------------------------------------------------

func (s *SQLiteStore) ListVulnerabilities(projectID string) ([]models.VulnerabilityNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id, project_id, title, cve, severity, cvss, description, remediation, status, evidence_refs, created_at FROM vulnerability_nodes WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.VulnerabilityNode
	for rows.Next() {
		var vn models.VulnerabilityNode
		var refs, ct string
		if err := rows.Scan(&vn.ID, &vn.ProjectID, &vn.Title, &vn.CVE, &vn.Severity, &vn.CVSS, &vn.Description, &vn.Remediation, &vn.Status, &refs, &ct); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(refs), &vn.EvidenceRefs)
		vn.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, vn)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CreateVulnerability(vn *models.VulnerabilityNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(vn.EvidenceRefs) == 0 {
		return fmt.Errorf("vulnerability must have at least one evidence reference")
	}
	if vn.ID == "" {
		vn.ID = genID("vuln")
	}
	vn.CreatedAt = time.Now()
	refs, _ := json.Marshal(emptySlice(vn.EvidenceRefs))
	_, err := s.db.Exec(`INSERT INTO vulnerability_nodes (id, project_id, title, cve, severity, cvss, description, remediation, status, evidence_refs, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		vn.ID, vn.ProjectID, vn.Title, vn.CVE, vn.Severity, vn.CVSS, vn.Description, vn.Remediation, vn.Status, string(refs), vn.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) UpdateVulnerability(vn *models.VulnerabilityNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	refs, _ := json.Marshal(emptySlice(vn.EvidenceRefs))
	_, err := s.db.Exec(`UPDATE vulnerability_nodes SET title=?, cve=?, severity=?, cvss=?, description=?, remediation=?, status=?, evidence_refs=? WHERE id=?`,
		vn.Title, vn.CVE, vn.Severity, vn.CVSS, vn.Description, vn.Remediation, vn.Status, string(refs), vn.ID)
	return err
}

// ---------------------------------------------------------------------------
// GraphEdge CRUD (GraphEdgeStore)
// ---------------------------------------------------------------------------

func (s *SQLiteStore) CreateEdge(e *models.GraphEdge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = genID("edge")
	}
	e.CreatedAt = time.Now()
	refs, _ := json.Marshal(emptySlice(e.EvidenceRefs))
	_, err := s.db.Exec(`INSERT INTO graph_edges (id, project_id, from_id, to_id, edge_type, evidence_refs, created_at) VALUES (?,?,?,?,?,?,?)`,
		e.ID, e.ProjectID, e.FromID, e.ToID, e.EdgeType, string(refs), e.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) ListEdges(projectID, fromID, toID string) ([]models.GraphEdge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query := `SELECT id, project_id, from_id, to_id, edge_type, evidence_refs, created_at FROM graph_edges WHERE project_id = ?`
	args := []interface{}{projectID}
	if fromID != "" {
		query += ` AND from_id = ?`
		args = append(args, fromID)
	}
	if toID != "" {
		query += ` AND to_id = ?`
		args = append(args, toID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.GraphEdge
	for rows.Next() {
		var e models.GraphEdge
		var refs, ct string
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.FromID, &e.ToID, &e.EdgeType, &refs, &ct); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(refs), &e.EvidenceRefs)
		e.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) DeleteEdge(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM graph_edges WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------------------
// Graph queries (OperationStore)
// ---------------------------------------------------------------------------

func (s *SQLiteStore) GraphQuery(projectID, startNodeID string, maxHops int) (*models.Subgraph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub := &models.Subgraph{Nodes: []map[string]any{}, Edges: []models.GraphEdge{}}
	visited := map[string]bool{}
	type qItem struct{ id string }
	queue := []qItem{{id: startNodeID}}
	visited[startNodeID] = true
	for hop := 0; hop <= maxHops && len(queue) > 0; hop++ {
		next := []qItem{}
		for _, cur := range queue {
			node := s.getNodeByID(cur.id)
			if node != nil {
				sub.Nodes = append(sub.Nodes, node)
			}
			rows, err := s.db.Query(`SELECT id, project_id, from_id, to_id, edge_type, evidence_refs, created_at FROM graph_edges WHERE project_id = ? AND (from_id = ? OR to_id = ?)`, projectID, cur.id, cur.id)
			if err != nil {
				continue
			}
			for rows.Next() {
				var e models.GraphEdge
				var refs, ct string
				rows.Scan(&e.ID, &e.ProjectID, &e.FromID, &e.ToID, &e.EdgeType, &refs, &ct)
				json.Unmarshal([]byte(refs), &e.EvidenceRefs)
				e.CreatedAt, _ = time.Parse(time.RFC3339, ct)
				sub.Edges = append(sub.Edges, e)
				neighbor := e.FromID
				if neighbor == cur.id {
					neighbor = e.ToID
				}
				if !visited[neighbor] {
					visited[neighbor] = true
					next = append(next, qItem{id: neighbor})
				}
			}
			rows.Close()
		}
		queue = next
	}
	return sub, nil
}

func (s *SQLiteStore) GraphTrace(projectID, nodeID string) (*models.TraceResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := &models.TraceResult{Chain: []models.TraceHop{}}
	current := nodeID
	for i := 0; i < 10; i++ {
		node := s.getNodeByID(current)
		if node == nil {
			break
		}
		nt, _ := node["node_type"].(string)
		label, _ := node["label"].(string)
		if label == "" {
			label, _ = node["title"].(string)
		}
		if label == "" {
			label, _ = node["name"].(string)
		}
		hop := models.TraceHop{NodeID: current, NodeType: nt, Label: label}
		row := s.db.QueryRow(`SELECT id, from_id, to_id, edge_type FROM graph_edges WHERE project_id = ? AND to_id = ? LIMIT 1`, projectID, current)
		var e models.GraphEdge
		if err := row.Scan(&e.ID, &e.FromID, &e.ToID, &e.EdgeType); err != nil {
			break
		}
		hop.EdgeType = e.EdgeType
		hop.FromID = e.FromID
		result.Chain = append(result.Chain, hop)
		current = e.FromID
	}
	return result, nil
}

func (s *SQLiteStore) getNodeByID(id string) map[string]any {
	tables := []struct {
		name   string
		label  string
	}{
		{"host_nodes", "hostname"},
		{"service_nodes", "name"},
		{"endpoint_nodes", "url"},
		{"session_nodes", "session_type"},
		{"evidence_nodes", "label"},
		{"hypothesis_nodes", "label"},
		{"vulnerability_nodes", "title"},
	}
	for _, tbl := range tables {
		row, err := s.db.Query(fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tbl.name), id)
		if err != nil {
			continue
		}
		cols, _ := row.Columns()
		if row.Next() {
			vals := make([]interface{}, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			row.Scan(ptrs...)
			m := map[string]any{"node_type": tbl.name}
			for i, c := range cols {
				m[c] = vals[i]
			}
			// ensure label field exists regardless of its actual column name
			if _, ok := m["label"]; !ok {
				if lbl, ok2 := m[tbl.label]; ok2 {
					m["label"] = lbl
				}
			}
			row.Close()
			return m
		}
		row.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func emptySlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}


// SearchAssets searches assets by full-text query across name, ips, domains, tech_stack.
func (s *SQLiteStore) SearchAssets(projectID, query string) ([]models.Asset, error) {
	like := "%" + query + "%"
	rows, err := s.db.Query(`SELECT id, project_id, name, ips, domains, tech_stack, scope, description, created_at FROM assets
		WHERE project_id=? AND (name LIKE ? OR ips LIKE ? OR domains LIKE ? OR tech_stack LIKE ? OR description LIKE ?)`,
		projectID, like, like, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Asset
	for rows.Next() {
		var a models.Asset
		var ct string
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Name, &a.IPs, &a.Domains, &a.TechStack, &a.Scope, &a.Description, &ct); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, a)
	}
	return out, rows.Err()
}

// SearchClues searches clues by query, optional type and status filters.
func (s *SQLiteStore) SearchClues(projectID, query, clueType, status string) ([]models.Clue, error) {
	cond := "project_id=?"
	args := []interface{}{projectID}
	if query != "" {
		cond += " AND (title LIKE ? OR content LIKE ?)"
		like := "%" + query + "%"
		args = append(args, like, like)
	}
	if clueType != "" {
		cond += " AND type=?"
		args = append(args, clueType)
	}
	if status != "" {
		cond += " AND status=?"
		args = append(args, status)
	}
	rows, err := s.db.Query(`SELECT id, project_id, title, content, type, status, created_at FROM clues WHERE `+cond, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Clue
	for rows.Next() {
		var c models.Clue
		var ct string
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Title, &c.Content, &c.Type, &c.Status, &ct); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		out = append(out, c)
	}
	return out, rows.Err()
}

// ProjectSummary returns a rollup of counts per entity type.
func (s *SQLiteStore) ProjectSummary(projectID string) (*models.ProjectSummary, error) {
	ps := &models.ProjectSummary{CluesByType: map[string]int{}}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM assets WHERE project_id=?", projectID).Scan(&ps.Assets); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM clues WHERE project_id=?", projectID).Scan(&ps.Clues); err != nil {
		return nil, err
	}
	rows, err := s.db.Query("SELECT type, COUNT(*) FROM clues WHERE project_id=? GROUP BY type", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		var c int
		if err := rows.Scan(&t, &c); err != nil {
			return nil, err
		}
		ps.CluesByType[t] = c
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM credentials WHERE project_id=?", projectID).Scan(&ps.Credentials); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM worklogs WHERE project_id=?", projectID).Scan(&ps.WorkLogs); err != nil {
		return nil, err
	}
	return ps, nil
}


// ensure the SQLite driver is referenced.
