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
