package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"adversarychef/nexus/internal/models"
)

// eventPayload is the JSON sent to the acasched webhook.
type eventPayload struct {
	ProjectID string `json:"project_id"`
	Action    string `json:"action"`
	Entity    string `json:"entity"`
	NodeID    string `json:"node_id"`
	ParentID  string `json:"parent_id"`
	Summary   string `json:"summary"`
	Timestamp string `json:"timestamp"`
}

// EventedStore wraps a Store and emits webhook events on every Create* call.
type EventedStore struct {
	inner      Store
	webhookURL string
	client     *http.Client
}

// NewEventedStore creates an EventedStore that delegates to inner and
// fires events to webhookURL (default http://127.0.0.1:9090/api/events).
func NewEventedStore(inner Store, webhookURL string) *EventedStore {
	if webhookURL == "" {
		webhookURL = "http://127.0.0.1:9090/api/events"
	}
	return &EventedStore{
		inner:      inner,
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

// emit sends an event payload to the webhook URL asynchronously.
func (es *EventedStore) emit(payload eventPayload) {
	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[evented] marshal event: %v", err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, es.webhookURL, bytes.NewReader(body))
		if err != nil {
			log.Printf("[evented] build request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := es.client.Do(req)
		if err != nil {
			log.Printf("[evented] POST %s: %v", es.webhookURL, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			log.Printf("[evented] POST %s returned %d", es.webhookURL, resp.StatusCode)
		}
	}()
}

func ts() string { return time.Now().UTC().Format(time.RFC3339) }

// ── Passthrough methods (delegate to inner) ──

func (es *EventedStore) ListProjects() ([]models.Project, error) {
	return es.inner.ListProjects()
}
func (es *EventedStore) GetProject(id string) (*models.Project, error) {
	return es.inner.GetProject(id)
}
func (es *EventedStore) UpdateProject(p *models.Project) error {
	return es.inner.UpdateProject(p)
}
func (es *EventedStore) DeleteProject(id string) error {
	return es.inner.DeleteProject(id)
}
func (es *EventedStore) ListAssets(projectID string) ([]models.Asset, error) {
	return es.inner.ListAssets(projectID)
}
func (es *EventedStore) GetAsset(id string) (*models.Asset, error) {
	return es.inner.GetAsset(id)
}
func (es *EventedStore) UpdateAsset(a *models.Asset) error {
	return es.inner.UpdateAsset(a)
}
func (es *EventedStore) DeleteAsset(id string) error {
	return es.inner.DeleteAsset(id)
}
func (es *EventedStore) ListClues(projectID string) ([]models.Clue, error) {
	return es.inner.ListClues(projectID)
}
func (es *EventedStore) GetClue(id string) (*models.Clue, error) {
	return es.inner.GetClue(id)
}
func (es *EventedStore) UpdateClue(c *models.Clue) error {
	return es.inner.UpdateClue(c)
}
func (es *EventedStore) DeleteClue(id string) error {
	return es.inner.DeleteClue(id)
}
func (es *EventedStore) ListCredentials(projectID string) ([]models.Credential, error) {
	return es.inner.ListCredentials(projectID)
}
func (es *EventedStore) GetCredential(id string) (*models.Credential, error) {
	return es.inner.GetCredential(id)
}
func (es *EventedStore) UpdateCredential(c *models.Credential) error {
	return es.inner.UpdateCredential(c)
}
func (es *EventedStore) DeleteCredential(id string) error {
	return es.inner.DeleteCredential(id)
}
func (es *EventedStore) ListWorkLogs(projectID string) ([]models.WorkLog, error) {
	return es.inner.ListWorkLogs(projectID)
}
func (es *EventedStore) GetWorkLog(id string) (*models.WorkLog, error) {
	return es.inner.GetWorkLog(id)
}
func (es *EventedStore) UpdateWorkLog(w *models.WorkLog) error {
	return es.inner.UpdateWorkLog(w)
}
func (es *EventedStore) DeleteWorkLog(id string) error {
	return es.inner.DeleteWorkLog(id)
}
func (es *EventedStore) SearchAssets(projectID, query string) ([]models.Asset, error) {
	return es.inner.SearchAssets(projectID, query)
}
func (es *EventedStore) SearchClues(projectID, query, clueType, status string) ([]models.Clue, error) {
	return es.inner.SearchClues(projectID, query, clueType, status)
}
func (es *EventedStore) ProjectSummary(projectID string) (*models.ProjectSummary, error) {
	return es.inner.ProjectSummary(projectID)
}

// ── OperationStore passthrough (list/get/update/delete) ──

func (es *EventedStore) ListHosts(projectID string) ([]models.HostNode, error) {
	return es.inner.ListHosts(projectID)
}
func (es *EventedStore) GetHost(id string) (*models.HostNode, error) {
	return es.inner.GetHost(id)
}
func (es *EventedStore) UpdateHost(h *models.HostNode) error {
	return es.inner.UpdateHost(h)
}
func (es *EventedStore) DeleteHost(id string) error {
	return es.inner.DeleteHost(id)
}
func (es *EventedStore) ListServices(projectID string) ([]models.ServiceNode, error) {
	return es.inner.ListServices(projectID)
}
func (es *EventedStore) GetService(id string) (*models.ServiceNode, error) {
	return es.inner.GetService(id)
}
func (es *EventedStore) UpdateService(s *models.ServiceNode) error {
	return es.inner.UpdateService(s)
}
func (es *EventedStore) DeleteService(id string) error {
	return es.inner.DeleteService(id)
}
func (es *EventedStore) ListEndpoints(projectID string) ([]models.EndpointNode, error) {
	return es.inner.ListEndpoints(projectID)
}
func (es *EventedStore) GetEndpoint(id string) (*models.EndpointNode, error) {
	return es.inner.GetEndpoint(id)
}
func (es *EventedStore) UpdateEndpointStatus(id, status, testedBy string) error {
	return es.inner.UpdateEndpointStatus(id, status, testedBy)
}
func (es *EventedStore) GetUntestedEndpoints(projectID string) ([]models.EndpointNode, error) {
	return es.inner.GetUntestedEndpoints(projectID)
}
func (es *EventedStore) ListSessions(projectID string) ([]models.SessionNode, error) {
	return es.inner.ListSessions(projectID)
}
func (es *EventedStore) GetSession(id string) (*models.SessionNode, error) {
	return es.inner.GetSession(id)
}
func (es *EventedStore) DeleteSession(id string) error {
	return es.inner.DeleteSession(id)
}
func (es *EventedStore) GraphQuery(projectID, startNodeID string, maxHops int) (*models.Subgraph, error) {
	return es.inner.GraphQuery(projectID, startNodeID, maxHops)
}
func (es *EventedStore) GraphTrace(projectID, nodeID string) (*models.TraceResult, error) {
	return es.inner.GraphTrace(projectID, nodeID)
}

// ── ReasoningStore passthrough (list/get/update) ──

func (es *EventedStore) ListEvidence(projectID string) ([]models.EvidenceNode, error) {
	return es.inner.ListEvidence(projectID)
}
func (es *EventedStore) ListHypotheses(projectID string) ([]models.HypothesisNode, error) {
	return es.inner.ListHypotheses(projectID)
}
func (es *EventedStore) UpdateHypothesisStatus(id, status string, confidence float64) error {
	return es.inner.UpdateHypothesisStatus(id, status, confidence)
}
func (es *EventedStore) ListVulnerabilities(projectID string) ([]models.VulnerabilityNode, error) {
	return es.inner.ListVulnerabilities(projectID)
}
func (es *EventedStore) UpdateVulnerability(v *models.VulnerabilityNode) error {
	return es.inner.UpdateVulnerability(v)
}

// ── GraphEdgeStore passthrough (list/delete) ──

func (es *EventedStore) ListEdges(projectID, fromID, toID string) ([]models.GraphEdge, error) {
	return es.inner.ListEdges(projectID, fromID, toID)
}
func (es *EventedStore) DeleteEdge(id string) error {
	return es.inner.DeleteEdge(id)
}

// ── Decorated Create* methods ──

func (es *EventedStore) CreateProject(p *models.Project) error {
	if err := es.inner.CreateProject(p); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: p.ID,
		Action:    "create",
		Entity:    "project",
		NodeID:    p.ID,
		ParentID:  "",
		Summary:   summaryProject(p),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateAsset(a *models.Asset) error {
	if err := es.inner.CreateAsset(a); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: a.ProjectID,
		Action:    "create",
		Entity:    "asset",
		NodeID:    a.ID,
		ParentID:  "",
		Summary:   summaryAsset(a),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateClue(c *models.Clue) error {
	if err := es.inner.CreateClue(c); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: c.ProjectID,
		Action:    "create",
		Entity:    "clue",
		NodeID:    c.ID,
		ParentID:  "",
		Summary:   summaryClue(c),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateCredential(c *models.Credential) error {
	if err := es.inner.CreateCredential(c); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: c.ProjectID,
		Action:    "create",
		Entity:    "credential",
		NodeID:    c.ID,
		ParentID:  "",
		Summary:   summaryCredential(c),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateWorkLog(w *models.WorkLog) error {
	if err := es.inner.CreateWorkLog(w); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: w.ProjectID,
		Action:    "create",
		Entity:    "worklog",
		NodeID:    w.ID,
		ParentID:  "",
		Summary:   summaryWorkLog(w),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateHost(h *models.HostNode) error {
	if err := es.inner.CreateHost(h); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: h.ProjectID,
		Action:    "create",
		Entity:    "host",
		NodeID:    h.ID,
		ParentID:  "",
		Summary:   summaryHost(h),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateService(s *models.ServiceNode) error {
	if err := es.inner.CreateService(s); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: s.ProjectID,
		Action:    "create",
		Entity:    "service",
		NodeID:    s.ID,
		ParentID:  s.HostID,
		Summary:   summaryService(s),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateEndpoint(e *models.EndpointNode) error {
	if err := es.inner.CreateEndpoint(e); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: e.ProjectID,
		Action:    "create",
		Entity:    "endpoint",
		NodeID:    e.ID,
		ParentID:  e.ServiceID,
		Summary:   summaryEndpoint(e),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateSession(s *models.SessionNode) error {
	if err := es.inner.CreateSession(s); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: s.ProjectID,
		Action:    "create",
		Entity:    "session",
		NodeID:    s.ID,
		ParentID:  "",
		Summary:   summarySession(s),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateEvidence(e *models.EvidenceNode) error {
	if err := es.inner.CreateEvidence(e); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: e.ProjectID,
		Action:    "create",
		Entity:    "evidence",
		NodeID:    e.ID,
		ParentID:  "",
		Summary:   summaryEvidence(e),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateHypothesis(h *models.HypothesisNode) error {
	if err := es.inner.CreateHypothesis(h); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: h.ProjectID,
		Action:    "create",
		Entity:    "hypothesis",
		NodeID:    h.ID,
		ParentID:  "",
		Summary:   summaryHypothesis(h),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateVulnerability(v *models.VulnerabilityNode) error {
	if err := es.inner.CreateVulnerability(v); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: v.ProjectID,
		Action:    "create",
		Entity:    "vuln",
		NodeID:    v.ID,
		ParentID:  "",
		Summary:   summaryVulnerability(v),
		Timestamp: ts(),
	})
	return nil
}

func (es *EventedStore) CreateEdge(e *models.GraphEdge) error {
	if err := es.inner.CreateEdge(e); err != nil {
		return err
	}
	es.emit(eventPayload{
		ProjectID: e.ProjectID,
		Action:    "create",
		Entity:    "edge",
		NodeID:    e.ID,
		ParentID:  "",
		Summary:   summaryEdge(e),
		Timestamp: ts(),
	})
	return nil
}

// ── Summary helpers ──

func summaryHost(h *models.HostNode) string {
	if len(h.IPs) > 0 {
		return h.IPs[0]
	}
	if h.Hostname != "" {
		return h.Hostname
	}
	return ""
}

func summaryService(s *models.ServiceNode) string {
	return fmt.Sprintf(":%d %s %s", s.Port, s.Name, s.Version)
}

func summaryEndpoint(e *models.EndpointNode) string {
	return fmt.Sprintf("%s %s", e.Method, e.URL)
}

func summaryEvidence(e *models.EvidenceNode) string {
	return fmt.Sprintf("%s (%s)", e.Label, e.Source)
}

func summaryVulnerability(v *models.VulnerabilityNode) string {
	return fmt.Sprintf("%s (%s)", v.Title, v.Severity)
}

func summarySession(s *models.SessionNode) string {
	return s.SessionType
}

func summaryHypothesis(h *models.HypothesisNode) string {
	return fmt.Sprintf("%s (%s)", h.Label, h.Status)
}

func summaryEdge(e *models.GraphEdge) string {
	return fmt.Sprintf("%s: %s->%s", e.EdgeType, e.FromID, e.ToID)
}

func summaryAsset(a *models.Asset) string {
	return a.Name
}

func summaryClue(c *models.Clue) string {
	return c.Title
}

func summaryCredential(c *models.Credential) string {
	return fmt.Sprintf("%s (%s)", c.Label, c.CredentialType)
}

func summaryWorkLog(w *models.WorkLog) string {
	return w.Title
}

func summaryProject(p *models.Project) string {
	return p.Name
}
