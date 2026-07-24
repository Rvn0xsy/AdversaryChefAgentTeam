package models
import "time"

// Project represents a penetration testing project.
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Asset represents a target asset.
type Asset struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	IPs         []string  `json:"ips,omitempty"`
	Domains     []string  `json:"domains,omitempty"`
	TechStack   []string  `json:"tech_stack,omitempty"`
	Scope       string    `json:"scope,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Clue represents a finding or clue discovered during testing.
type Clue struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content,omitempty"`
	Type      string    `json:"type,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Credential represents an account credential.
type Credential struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	AssetID        string    `json:"asset_id,omitempty"`
	CredentialType string    `json:"credential_type"`
	Label          string    `json:"label"`
	Value          string    `json:"value"`
	ExpiresAt      string    `json:"expires_at,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// WorkLog represents a work log entry.
type WorkLog struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
