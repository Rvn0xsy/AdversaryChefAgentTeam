package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/models"
	"adversarychef/nexus/internal/store"
	"adversarychef/mcputil"
)

// registerReasoningNodes registers tools for Reasoning Graph nodes
// (evidence, hypothesis, vulnerability).
func registerReasoningNodes(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	registerEvidenceTools(server, s, sm)
	registerHypothesisTools(server, s, sm)
	registerVulnerabilityTools(server, s, sm)
}

// ── Evidence tools ──

func registerEvidenceTools(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// evidence_list
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "evidence_list",
		Description: "List evidence nodes by project ID",
	}, scopedLister(sm, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID string `json:"project_id" jsonschema:"Project ID"`
	}) (*mcp.CallToolResult, any, error) {
		items, err := s.ListEvidence(params.ProjectID)
		if err != nil {
			return mcputil.TextResult("query failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(items)
		return mcputil.TextResult(string(b)), nil, nil
	}))

	// evidence_create
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "evidence_create",
		Description: "Create an evidence node (e.g., scan result, tool output reference)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID  string `json:"project_id" jsonschema:"Project ID"`
		Label      string `json:"label" jsonschema:"Evidence label"`
		Source     string `json:"source" jsonschema:"Evidence source (e.g. nmap, nuclei)"`
		ContentRef string `json:"content_ref,omitempty" jsonschema:"Reference to external content"`
	}) (*mcp.CallToolResult, any, error) {
		e := &models.EvidenceNode{
			ProjectID:  params.ProjectID,
			Label:      params.Label,
			Source:     params.Source,
			ContentRef: params.ContentRef,
		}
		if err := s.CreateEvidence(e); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(e)
		return mcputil.TextResult(string(b)), nil, nil
	})
}

// ── Hypothesis tools ──

func registerHypothesisTools(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// hypothesis_list
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "hypothesis_list",
		Description: "List hypothesis nodes by project ID",
	}, scopedLister(sm, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID string `json:"project_id" jsonschema:"Project ID"`
	}) (*mcp.CallToolResult, any, error) {
		items, err := s.ListHypotheses(params.ProjectID)
		if err != nil {
			return mcputil.TextResult("query failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(items)
		return mcputil.TextResult(string(b)), nil, nil
	}))

	// hypothesis_create
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "hypothesis_create",
		Description: "Create a hypothesis node",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID  string  `json:"project_id" jsonschema:"Project ID"`
		Label      string  `json:"label" jsonschema:"Hypothesis label"`
		Confidence float64 `json:"confidence" jsonschema:"Confidence score (0.0-1.0)"`
	}) (*mcp.CallToolResult, any, error) {
		h := &models.HypothesisNode{
			ProjectID:  params.ProjectID,
			Label:      params.Label,
			Confidence: params.Confidence,
			Status:     "proposed",
		}
		if err := s.CreateHypothesis(h); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(h)
		return mcputil.TextResult(string(b)), nil, nil
	})

	// hypothesis_update
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "hypothesis_update",
		Description: "Update hypothesis status and confidence",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ID         string  `json:"id" jsonschema:"Hypothesis ID"`
		Status     string  `json:"status" jsonschema:"Status (proposed|confirmed|rejected)"`
		Confidence float64 `json:"confidence" jsonschema:"Confidence score (0.0-1.0)"`
	}) (*mcp.CallToolResult, any, error) {
		if err := s.UpdateHypothesisStatus(params.ID, params.Status, params.Confidence); err != nil {
			return mcputil.TextResult("update failed: " + err.Error()), nil, nil
		}
		return mcputil.TextResult("updated"), nil, nil
	})
}

// ── Vulnerability tools ──

func registerVulnerabilityTools(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// vulnerability_list
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "vulnerability_list",
		Description: "List vulnerability nodes by project ID",
	}, scopedLister(sm, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID string `json:"project_id" jsonschema:"Project ID"`
	}) (*mcp.CallToolResult, any, error) {
		items, err := s.ListVulnerabilities(params.ProjectID)
		if err != nil {
			return mcputil.TextResult("query failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(items)
		return mcputil.TextResult(string(b)), nil, nil
	}))

	// vulnerability_create
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "vulnerability_create",
		Description: "Create a vulnerability node (REQUIRES evidence_refs with at least one entry)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID    string   `json:"project_id" jsonschema:"Project ID"`
		Title        string   `json:"title" jsonschema:"Vulnerability title"`
		CVE          string   `json:"cve,omitempty" jsonschema:"CVE identifier"`
		Severity     string   `json:"severity" jsonschema:"Severity (critical|high|medium|low|info)"`
		CVSS         float64  `json:"cvss,omitempty" jsonschema:"CVSS score (0.0-10.0)"`
		Description  string   `json:"description,omitempty" jsonschema:"Vulnerability description"`
		Remediation  string   `json:"remediation,omitempty" jsonschema:"Remediation guidance"`
		EvidenceRefs []string `json:"evidence_refs" jsonschema:"Evidence node IDs (REQUIRED, minimum 1)"`
	}) (*mcp.CallToolResult, any, error) {
		if len(params.EvidenceRefs) == 0 {
			return mcputil.TextResult("evidence_refs is required"), nil, nil
		}
		v := &models.VulnerabilityNode{
			ProjectID:    params.ProjectID,
			Title:        params.Title,
			CVE:          params.CVE,
			Severity:     params.Severity,
			CVSS:         params.CVSS,
			Description:  params.Description,
			Remediation:  params.Remediation,
			Status:       "open",
			EvidenceRefs: params.EvidenceRefs,
		}
		if err := s.CreateVulnerability(v); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(v)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
