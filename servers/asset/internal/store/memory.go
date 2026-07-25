package store
import (
	"fmt"
	"strings"
	"sync"
	"time"

	"adversarychef/asset/internal/models"
)

// MemoryStore is an in-memory store for development use.
type MemoryStore struct {
	mu          sync.RWMutex
	projects    map[string]*models.Project
	assets      map[string]*models.Asset
	clues       map[string]*models.Clue
	credentials map[string]*models.Credential
	worklogs    map[string]*models.WorkLog
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		projects:    make(map[string]*models.Project),
		assets:      make(map[string]*models.Asset),
		clues:       make(map[string]*models.Clue),
		credentials: make(map[string]*models.Credential),
		worklogs:    make(map[string]*models.WorkLog),
	}
}

func genID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func (s *MemoryStore) ListProjects() ([]models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Project, 0, len(s.projects))
	for _, p := range s.projects {
		out = append(out, *p)
	}
	return out, nil
}

func (s *MemoryStore) GetProject(id string) (*models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[id]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	return p, nil
}

func (s *MemoryStore) CreateProject(p *models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = genID("proj")
	}
	p.CreatedAt = time.Now()
	s.projects[p.ID] = p
	return nil
}

func (s *MemoryStore) UpdateProject(p *models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[p.ID]; !ok {
		return fmt.Errorf("project not found: %s", p.ID)
	}
	s.projects[p.ID] = p
	return nil
}

func (s *MemoryStore) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.projects, id)
	return nil
}

func (s *MemoryStore) ListAssets(projectID string) ([]models.Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Asset, 0)
	for _, a := range s.assets {
		if a.ProjectID == projectID {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetAsset(id string) (*models.Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.assets[id]
	if !ok {
		return nil, fmt.Errorf("asset not found: %s", id)
	}
	return a, nil
}

func (s *MemoryStore) CreateAsset(a *models.Asset) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.ID == "" {
		a.ID = genID("asset")
	}
	a.CreatedAt = time.Now()
	s.assets[a.ID] = a
	return nil
}

func (s *MemoryStore) UpdateAsset(a *models.Asset) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.assets[a.ID]; !ok {
		return fmt.Errorf("asset not found: %s", a.ID)
	}
	s.assets[a.ID] = a
	return nil
}

func (s *MemoryStore) DeleteAsset(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.assets, id)
	return nil
}

func (s *MemoryStore) ListClues(projectID string) ([]models.Clue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Clue, 0)
	for _, c := range s.clues {
		if c.ProjectID == projectID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetClue(id string) (*models.Clue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clues[id]
	if !ok {
		return nil, fmt.Errorf("clue not found: %s", id)
	}
	return c, nil
}

func (s *MemoryStore) CreateClue(c *models.Clue) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = genID("clue")
	}
	c.CreatedAt = time.Now()
	s.clues[c.ID] = c
	return nil
}

func (s *MemoryStore) UpdateClue(c *models.Clue) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clues[c.ID]; !ok {
		return fmt.Errorf("clue not found: %s", c.ID)
	}
	s.clues[c.ID] = c
	return nil
}

func (s *MemoryStore) DeleteClue(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clues, id)
	return nil
}

func (s *MemoryStore) ListCredentials(projectID string) ([]models.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Credential, 0)
	for _, c := range s.credentials {
		if c.ProjectID == projectID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetCredential(id string) (*models.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.credentials[id]
	if !ok {
		return nil, fmt.Errorf("credential not found: %s", id)
	}
	return c, nil
}

func (s *MemoryStore) CreateCredential(c *models.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = genID("cred")
	}
	c.CreatedAt = time.Now()
	s.credentials[c.ID] = c
	return nil
}

func (s *MemoryStore) UpdateCredential(c *models.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.credentials[c.ID]; !ok {
		return fmt.Errorf("credential not found: %s", c.ID)
	}
	s.credentials[c.ID] = c
	return nil
}

func (s *MemoryStore) DeleteCredential(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.credentials, id)
	return nil
}

func (s *MemoryStore) ListWorkLogs(projectID string) ([]models.WorkLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.WorkLog, 0)
	for _, w := range s.worklogs {
		if w.ProjectID == projectID {
			out = append(out, *w)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetWorkLog(id string) (*models.WorkLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.worklogs[id]
	if !ok {
		return nil, fmt.Errorf("worklog not found: %s", id)
	}
	return w, nil
}

func (s *MemoryStore) CreateWorkLog(w *models.WorkLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w.ID == "" {
		w.ID = genID("wl")
	}
	w.CreatedAt = time.Now()
	s.worklogs[w.ID] = w
	return nil
}

func (s *MemoryStore) UpdateWorkLog(w *models.WorkLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.worklogs[w.ID]; !ok {
		return fmt.Errorf("worklog not found: %s", w.ID)
	}
	s.worklogs[w.ID] = w
	return nil
}

func (s *MemoryStore) DeleteWorkLog(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.worklogs, id)
	return nil
}

func (s *MemoryStore) SearchAssets(projectID, query string) ([]models.Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if query == "" {
		return nil, nil
	}
	lower := strings.ToLower(query)
	var result []models.Asset
	for _, a := range s.assets {
		if a.ProjectID != projectID {
			continue
		}
		if contains(lower, a.Name, a.Description) || containsAny(lower, a.IPs) || containsAny(lower, a.Domains) || containsAny(lower, a.TechStack) {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (s *MemoryStore) SearchClues(projectID, query, clueType, status string) ([]models.Clue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lower := strings.ToLower(query)
	var result []models.Clue
	for _, c := range s.clues {
		if c.ProjectID != projectID {
			continue
		}
		if clueType != "" && c.Type != clueType {
			continue
		}
		if status != "" && c.Status != status {
			continue
		}
		if lower == "" || contains(lower, c.Title, c.Content) {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (s *MemoryStore) ProjectSummary(projectID string) (*models.ProjectSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ps := &models.ProjectSummary{CluesByType: map[string]int{}}
	for _, a := range s.assets {
		if a.ProjectID == projectID {
			ps.Assets++
		}
	}
	for _, c := range s.clues {
		if c.ProjectID == projectID {
			ps.Clues++
			ps.CluesByType[c.Type]++
		}
	}
	for _, cr := range s.credentials {
		if cr.ProjectID == projectID {
			ps.Credentials++
		}
	}
	for _, w := range s.worklogs {
		if w.ProjectID == projectID {
			ps.WorkLogs++
		}
	}
	return ps, nil
}

func contains(lower string, values ...string) bool {
	for _, v := range values {
		if strings.Contains(strings.ToLower(v), lower) {
			return true
		}
	}
	return false
}

func containsAny(lower string, values []string) bool {
	for _, v := range values {
		if strings.Contains(strings.ToLower(v), lower) {
			return true
		}
	}
	return false
}
