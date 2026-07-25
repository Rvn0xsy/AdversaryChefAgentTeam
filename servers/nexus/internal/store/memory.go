package store
import (
	"fmt"
	"strings"
	"sync"
	"time"

	"adversarychef/nexus/internal/models"
)

// MemoryStore is an in-memory store for development use.
type MemoryStore struct {
	mu          sync.RWMutex
	projects    map[string]*models.Project
	assets      map[string]*models.Asset
	clues       map[string]*models.Clue
	credentials map[string]*models.Credential
	worklogs    map[string]*models.WorkLog

	// Graph tables
	hosts            map[string]*models.HostNode
	services         map[string]*models.ServiceNode
	endpoints        map[string]*models.EndpointNode
	sessions         map[string]*models.SessionNode
	evidenceNodes    map[string]*models.EvidenceNode
	hypotheses       map[string]*models.HypothesisNode
	vulnerabilities  map[string]*models.VulnerabilityNode
	edges            map[string]*models.GraphEdge
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		projects:        make(map[string]*models.Project),
		assets:          make(map[string]*models.Asset),
		clues:           make(map[string]*models.Clue),
		credentials:     make(map[string]*models.Credential),
		worklogs:        make(map[string]*models.WorkLog),
		hosts:           make(map[string]*models.HostNode),
		services:        make(map[string]*models.ServiceNode),
		endpoints:       make(map[string]*models.EndpointNode),
		sessions:        make(map[string]*models.SessionNode),
		evidenceNodes:   make(map[string]*models.EvidenceNode),
		hypotheses:      make(map[string]*models.HypothesisNode),
		vulnerabilities: make(map[string]*models.VulnerabilityNode),
		edges:           make(map[string]*models.GraphEdge),
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

// ---------------------------------------------------------------------------
// Host stubs (OperationStore)
// ---------------------------------------------------------------------------

func (s *MemoryStore) ListHosts(projectID string) ([]models.HostNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.HostNode, 0)
	for _, h := range s.hosts {
		if h.ProjectID == projectID {
			out = append(out, *h)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetHost(id string) (*models.HostNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.hosts[id]
	if !ok {
		return nil, fmt.Errorf("host not found: %s", id)
	}
	return h, nil
}

func (s *MemoryStore) CreateHost(h *models.HostNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h.ID == "" {
		h.ID = genID("host")
	}
	h.CreatedAt = time.Now()
	s.hosts[h.ID] = h
	return nil
}

func (s *MemoryStore) UpdateHost(h *models.HostNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.hosts[h.ID]; !ok {
		return fmt.Errorf("host not found: %s", h.ID)
	}
	s.hosts[h.ID] = h
	return nil
}

func (s *MemoryStore) DeleteHost(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.hosts, id)
	return nil
}

// ---------------------------------------------------------------------------
// Service stubs (OperationStore)
// ---------------------------------------------------------------------------

func (s *MemoryStore) ListServices(projectID string) ([]models.ServiceNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.ServiceNode, 0)
	for _, sv := range s.services {
		if sv.ProjectID == projectID {
			out = append(out, *sv)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetService(id string) (*models.ServiceNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sv, ok := s.services[id]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", id)
	}
	return sv, nil
}

func (s *MemoryStore) CreateService(sv *models.ServiceNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sv.ID == "" {
		sv.ID = genID("svc")
	}
	sv.CreatedAt = time.Now()
	s.services[sv.ID] = sv
	return nil
}

func (s *MemoryStore) UpdateService(sv *models.ServiceNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[sv.ID]; !ok {
		return fmt.Errorf("service not found: %s", sv.ID)
	}
	s.services[sv.ID] = sv
	return nil
}

func (s *MemoryStore) DeleteService(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.services, id)
	return nil
}

// ---------------------------------------------------------------------------
// Endpoint stubs (OperationStore)
// ---------------------------------------------------------------------------

func (s *MemoryStore) ListEndpoints(projectID string) ([]models.EndpointNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.EndpointNode, 0)
	for _, ep := range s.endpoints {
		if ep.ProjectID == projectID {
			out = append(out, *ep)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetEndpoint(id string) (*models.EndpointNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ep, ok := s.endpoints[id]
	if !ok {
		return nil, fmt.Errorf("endpoint not found: %s", id)
	}
	return ep, nil
}

func (s *MemoryStore) CreateEndpoint(ep *models.EndpointNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ep.ID == "" {
		ep.ID = genID("ep")
	}
	ep.CreatedAt = time.Now()
	s.endpoints[ep.ID] = ep
	return nil
}

func (s *MemoryStore) UpdateEndpointStatus(id, status, testedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep, ok := s.endpoints[id]
	if !ok {
		return fmt.Errorf("endpoint not found: %s", id)
	}
	ep.Status = status
	ep.TestedBy = testedBy
	return nil
}

func (s *MemoryStore) GetUntestedEndpoints(projectID string) ([]models.EndpointNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.EndpointNode, 0)
	for _, ep := range s.endpoints {
		if ep.ProjectID == projectID && ep.Status == "discovered" {
			out = append(out, *ep)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Session stubs (OperationStore)
// ---------------------------------------------------------------------------

func (s *MemoryStore) ListSessions(projectID string) ([]models.SessionNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.SessionNode, 0)
	for _, sn := range s.sessions {
		if sn.ProjectID == projectID {
			out = append(out, *sn)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetSession(id string) (*models.SessionNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sn, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return sn, nil
}

func (s *MemoryStore) CreateSession(sn *models.SessionNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sn.ID == "" {
		sn.ID = genID("sess")
	}
	sn.CreatedAt = time.Now()
	s.sessions[sn.ID] = sn
	return nil
}

func (s *MemoryStore) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

// ---------------------------------------------------------------------------
// Evidence stubs (ReasoningStore)
// ---------------------------------------------------------------------------

func (s *MemoryStore) ListEvidence(projectID string) ([]models.EvidenceNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.EvidenceNode, 0)
	for _, ev := range s.evidenceNodes {
		if ev.ProjectID == projectID {
			out = append(out, *ev)
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateEvidence(ev *models.EvidenceNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev.ID == "" {
		ev.ID = genID("evid")
	}
	ev.CreatedAt = time.Now()
	s.evidenceNodes[ev.ID] = ev
	return nil
}

// ---------------------------------------------------------------------------
// Hypothesis stubs (ReasoningStore)
// ---------------------------------------------------------------------------

func (s *MemoryStore) ListHypotheses(projectID string) ([]models.HypothesisNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.HypothesisNode, 0)
	for _, hy := range s.hypotheses {
		if hy.ProjectID == projectID {
			out = append(out, *hy)
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateHypothesis(hy *models.HypothesisNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hy.ID == "" {
		hy.ID = genID("hyp")
	}
	hy.CreatedAt = time.Now()
	s.hypotheses[hy.ID] = hy
	return nil
}

func (s *MemoryStore) UpdateHypothesisStatus(id, status string, confidence float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hy, ok := s.hypotheses[id]
	if !ok {
		return fmt.Errorf("hypothesis not found: %s", id)
	}
	hy.Status = status
	hy.Confidence = confidence
	return nil
}

// ---------------------------------------------------------------------------
// Vulnerability stubs (ReasoningStore)
// ---------------------------------------------------------------------------

func (s *MemoryStore) ListVulnerabilities(projectID string) ([]models.VulnerabilityNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.VulnerabilityNode, 0)
	for _, vn := range s.vulnerabilities {
		if vn.ProjectID == projectID {
			out = append(out, *vn)
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateVulnerability(vn *models.VulnerabilityNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(vn.EvidenceRefs) == 0 {
		return fmt.Errorf("vulnerability must have at least one evidence reference")
	}
	if vn.ID == "" {
		vn.ID = genID("vuln")
	}
	vn.CreatedAt = time.Now()
	s.vulnerabilities[vn.ID] = vn
	return nil
}

func (s *MemoryStore) UpdateVulnerability(vn *models.VulnerabilityNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.vulnerabilities[vn.ID]; !ok {
		return fmt.Errorf("vulnerability not found: %s", vn.ID)
	}
	s.vulnerabilities[vn.ID] = vn
	return nil
}

// ---------------------------------------------------------------------------
// GraphEdge stubs (GraphEdgeStore)
// ---------------------------------------------------------------------------

func (s *MemoryStore) CreateEdge(e *models.GraphEdge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == "" {
		e.ID = genID("edge")
	}
	e.CreatedAt = time.Now()
	s.edges[e.ID] = e
	return nil
}

func (s *MemoryStore) ListEdges(projectID, fromID, toID string) ([]models.GraphEdge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.GraphEdge, 0)
	for _, e := range s.edges {
		if e.ProjectID != projectID {
			continue
		}
		if fromID != "" && e.FromID != fromID {
			continue
		}
		if toID != "" && e.ToID != toID {
			continue
		}
		out = append(out, *e)
	}
	return out, nil
}

func (s *MemoryStore) DeleteEdge(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.edges, id)
	return nil
}

// ---------------------------------------------------------------------------
// GraphQuery / GraphTrace stubs (not-implemented for memory store)
// ---------------------------------------------------------------------------

func (s *MemoryStore) GraphQuery(projectID, startNodeID string, maxHops int) (*models.Subgraph, error) {
	return nil, fmt.Errorf("GraphQuery: not implemented for in-memory store")
}

func (s *MemoryStore) GraphTrace(projectID, nodeID string) (*models.TraceResult, error) {
	return nil, fmt.Errorf("GraphTrace: not implemented for in-memory store")
}
