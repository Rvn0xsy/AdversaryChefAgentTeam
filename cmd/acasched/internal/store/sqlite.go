package store

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Task struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	ParentID     string     `json:"parent_id"`
	Agent        string     `json:"agent"`
	Status       string     `json:"status"` // pending|dispatched|running|done|failed|timeout|skipped
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Result       string     `json:"result"`
	Error        string     `json:"error"`
	CreatedBy    string     `json:"created_by"`
	MaxTurns     int        `json:"max_turns"`
	TimeoutSecs  int        `json:"timeout_secs"`
	RetryCount   int        `json:"retry_count"`
	Attempt      int        `json:"attempt"`
	CreatedAt    time.Time  `json:"created_at"`
	DispatchedAt *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}


type Store struct {
	mu sync.RWMutex
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("wal: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			status TEXT DEFAULT 'active',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			parent_id TEXT,
			agent TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			result TEXT DEFAULT '',
			error TEXT DEFAULT '',
			created_by TEXT NOT NULL,
			max_turns INTEGER DEFAULT 40,
			timeout_secs INTEGER DEFAULT 1800,
			retry_count INTEGER DEFAULT 1,
			attempt INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			dispatched_at TEXT,
			completed_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(project_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_id)`,
	}
	for _, d := range ddl {
		if _, err := s.db.Exec(d); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateTask(t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t.ID == "" {
		t.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}
	t.CreatedAt = time.Now()

	_, err := s.db.Exec(
		`INSERT INTO tasks (id, project_id, parent_id, agent, status, title, description, created_by, max_turns, timeout_secs, retry_count, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.ProjectID, t.ParentID, t.Agent, t.Status, t.Title, t.Description, t.CreatedBy, t.MaxTurns, t.TimeoutSecs, t.RetryCount, t.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *Store) ListPending(projectID string) ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows *sql.Rows
	var err error
	if projectID == "" {
		rows, err = s.db.Query(
			`SELECT id, project_id, parent_id, agent, status, title, description, result, error, created_by, max_turns, timeout_secs, retry_count, attempt, created_at FROM tasks WHERE status = 'pending' ORDER BY created_at ASC`,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, project_id, parent_id, agent, status, title, description, result, error, created_by, max_turns, timeout_secs, retry_count, attempt, created_at FROM tasks WHERE project_id = ? AND status = 'pending' ORDER BY created_at ASC`,
			projectID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		var ct string
		var dt, cmt sql.NullString

		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Agent, &t.Status, &t.Title, &t.Description, &t.Result, &t.Error, &t.CreatedBy, &t.MaxTurns, &t.TimeoutSecs, &t.RetryCount, &t.Attempt, &ct); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		if dt.Valid {
			pt, _ := time.Parse(time.RFC3339, dt.String)
			t.DispatchedAt = &pt
		}
		if cmt.Valid {
			pt, _ := time.Parse(time.RFC3339, cmt.String)
			t.CompletedAt = &pt
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) UpdateStatus(id, status, result, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	_, e := s.db.Exec(`UPDATE tasks SET status=?, result=?, error=?, completed_at=? WHERE id=?`, status, result, errMsg, now, id)
	return e
}

func (s *Store) MarkDispatched(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE tasks SET status='dispatched', dispatched_at=? WHERE id=?`, now, id)
	return err
}

func (s *Store) MarkRunning(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`UPDATE tasks SET status='running' WHERE id=?`, id)
	return err
}

func (s *Store) FindChildren(parentID string) ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, status FROM tasks WHERE parent_id = ?`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Status); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTask(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var t Task
	var ct string

	err := s.db.QueryRow(
		`SELECT id, project_id, parent_id, agent, status, title, description, result, error, created_by, max_turns, timeout_secs, retry_count, attempt, created_at FROM tasks WHERE id = ?`,
		id,
	).Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Agent, &t.Status, &t.Title, &t.Description, &t.Result, &t.Error, &t.CreatedBy, &t.MaxTurns, &t.TimeoutSecs, &t.RetryCount, &t.Attempt, &ct)
	if err != nil {
		return nil, err
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &t, nil
}

func (s *Store) CreateProject(p *Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.ID == "" {
		p.ID = fmt.Sprintf("proj_%d", time.Now().UnixNano())
	}
	p.CreatedAt = time.Now()

	_, err := s.db.Exec(`INSERT INTO projects (id, name, description, status, created_at) VALUES (?,?,?,?,?)`,
		p.ID, p.Name, p.Description, p.Status, p.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *Store) GetProject(id string) (*Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var p Project
	var ct string

	err := s.db.QueryRow(`SELECT id, name, description, status, created_at FROM projects WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Status, &ct)
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, ct)
	return &p, nil
}

func (s *Store) ListProjects() ([]Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, name, description, status, created_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		var ct string
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &ct); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *Store) UpdateProject(id, name, description, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := "UPDATE projects SET name=?, description=?, status=? WHERE id=?"
	_, err := s.db.Exec(query, name, description, status, id)
	return err
}

// ListTasks returns all tasks for a project, optionally filtered by status.
// statusFilter is a comma-separated list like "pending,running". Empty means all.
func (s *Store) ListTasks(projectID string, statusFilter string) ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows *sql.Rows
	var err error

	if statusFilter == "" {
		rows, err = s.db.Query(
			`SELECT id, project_id, parent_id, agent, status, title, description, result, error, created_by, max_turns, timeout_secs, retry_count, attempt, created_at FROM tasks WHERE project_id = ? ORDER BY created_at ASC`,
			projectID)
	} else {
		// Build IN clause: status IN ('pending','running')
		statuses := strings.Split(statusFilter, ",")
		placeholders := make([]string, len(statuses))
		args := make([]any, 0, len(statuses)+1)
		args = append(args, projectID)
		for i, st := range statuses {
			placeholders[i] = "?"
			args = append(args, strings.TrimSpace(st))
		}
		query := fmt.Sprintf(
			`SELECT id, project_id, parent_id, agent, status, title, description, result, error, created_by, max_turns, timeout_secs, retry_count, attempt, created_at FROM tasks WHERE project_id = ? AND status IN (%s) ORDER BY created_at ASC`,
			strings.Join(placeholders, ","))
		rows, err = s.db.Query(query, args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var ct string
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Agent, &t.Status, &t.Title, &t.Description, &t.Result, &t.Error, &t.CreatedBy, &t.MaxTurns, &t.TimeoutSecs, &t.RetryCount, &t.Attempt, &ct); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *Store) ListTasksByProject(projectID string) ([]Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		`SELECT id, project_id, parent_id, agent, status, title, description, result, error, created_by, max_turns, timeout_secs, retry_count, attempt, created_at FROM tasks WHERE project_id = ? ORDER BY created_at ASC`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var ct string
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.Agent, &t.Status, &t.Title, &t.Description, &t.Result, &t.Error, &t.CreatedBy, &t.MaxTurns, &t.TimeoutSecs, &t.RetryCount, &t.Attempt, &ct); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, ct)
		tasks = append(tasks, t)
	}
	return tasks, nil
}
