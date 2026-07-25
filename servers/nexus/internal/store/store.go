package store
import "adversarychef/nexus/internal/models"

// Store is the data persistence interface; can be swapped for Postgres.
type Store interface {
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

	// Embedded graph sub-interfaces
	OperationStore
	ReasoningStore
	GraphEdgeStore
}

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
