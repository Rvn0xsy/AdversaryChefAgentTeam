package store
import "adversarychef/asset/internal/models"

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
}
