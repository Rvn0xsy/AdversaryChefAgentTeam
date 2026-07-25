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
