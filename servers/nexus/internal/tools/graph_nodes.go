package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/nexus/internal/models"
	"adversarychef/nexus/internal/store"
	"adversarychef/mcputil"
)

// ── Parameter types ──

type hostParams struct {
	ProjectID string   `json:"project_id" jsonschema:"Project ID"`
	IPs       []string `json:"ips" jsonschema:"IP addresses"`
	Hostname  string   `json:"hostname,omitempty" jsonschema:"Hostname"`
	OS        string   `json:"os,omitempty" jsonschema:"Operating system"`
}

type serviceParams struct {
	ProjectID string `json:"project_id" jsonschema:"Project ID"`
	HostID    string `json:"host_id" jsonschema:"Host node ID"`
	Port      int    `json:"port" jsonschema:"Port number"`
	Protocol  string `json:"protocol,omitempty" jsonschema:"Protocol (tcp/udp)"`
	Name      string `json:"name" jsonschema:"Service name"`
	Version   string `json:"version,omitempty" jsonschema:"Version"`
}

type endpointParams struct {
	ProjectID   string   `json:"project_id" jsonschema:"Project ID"`
	ServiceID   string   `json:"service_id" jsonschema:"Service node ID"`
	URL         string   `json:"url" jsonschema:"URL path"`
	Method      string   `json:"method,omitempty" jsonschema:"HTTP method"`
	Parameters  []string `json:"parameters,omitempty" jsonschema:"Parameter names"`
	DiscoveredBy string  `json:"discovered_by,omitempty" jsonschema:"Discovering agent name"`
}

type endpointUpdateStatusParams struct {
	ID       string `json:"id" jsonschema:"Endpoint ID"`
	Status   string `json:"status" jsonschema:"New status (tested|vulnerable|safe)"`
	TestedBy string `json:"tested_by,omitempty" jsonschema:"Agent that performed the test"`
}

type sessionParams struct {
	ProjectID   string `json:"project_id" jsonschema:"Project ID"`
	SessionType string `json:"session_type" jsonschema:"Session type (web/c2_shell/ssh/tunnel)"`
	CreatedBy   string `json:"created_by" jsonschema:"Creating agent name"`
	URL         string `json:"url,omitempty" jsonschema:"Session URL"`
	AssetID     string `json:"asset_id,omitempty" jsonschema:"Related asset ID"`
}

// ── scopedLister wraps a handler with session binding validation.
// Full MCP session context wiring is deferred to Task 8.
func scopedLister[P any](sm *mcputil.SessionMap, handler func(context.Context, *mcp.CallToolRequest, P) (*mcp.CallToolResult, any, error)) func(context.Context, *mcp.CallToolRequest, P) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, params P) (*mcp.CallToolResult, any, error) {
		// Session binding validation deferred to Task 8.
		return handler(ctx, req, params)
	}
}

// ── Registration ──

func registerGraphNodes(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	registerHostTools(server, s, sm)
	registerServiceTools(server, s, sm)
	registerEndpointTools(server, s, sm)
	registerSessionTools(server, s, sm)
}

// ── Host tools ──

func registerHostTools(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// host_list
	Lister(server, "host_list", "List all host nodes by project ID", "Host", s.ListHosts)

	// host_get
	Getter(server, "host_get", "Get a host node by ID", "Host", s.GetHost)

	// host_create
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "host_create",
		Description: "Create a new host node",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params hostParams) (*mcp.CallToolResult, any, error) {
		h := &models.HostNode{
			ProjectID: params.ProjectID,
			IPs:       params.IPs,
			Hostname:  params.Hostname,
			OS:        params.OS,
		}
		if err := s.CreateHost(h); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(h)
		return mcputil.TextResult(string(b)), nil, nil
	})
}

// ── Service tools ──

func registerServiceTools(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// service_list
	Lister(server, "service_list", "List all service nodes by project ID", "Service", s.ListServices)

	// service_get
	Getter(server, "service_get", "Get a service node by ID", "Service", s.GetService)

	// service_create
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "service_create",
		Description: "Create a new service node",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params serviceParams) (*mcp.CallToolResult, any, error) {
		svc := &models.ServiceNode{
			ProjectID: params.ProjectID,
			HostID:    params.HostID,
			Port:      params.Port,
			Protocol:  params.Protocol,
			Name:      params.Name,
			Version:   params.Version,
		}
		if err := s.CreateService(svc); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(svc)
		return mcputil.TextResult(string(b)), nil, nil
	})
}

// ── Endpoint tools ──

func registerEndpointTools(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// endpoint_list
	Lister(server, "endpoint_list", "List all endpoint nodes by project ID", "Endpoint", s.ListEndpoints)

	// endpoint_get
	Getter(server, "endpoint_get", "Get an endpoint node by ID", "Endpoint", s.GetEndpoint)

	// endpoint_create
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "endpoint_create",
		Description: "Create a new endpoint node (auto-dedup by project_id+url+method)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params endpointParams) (*mcp.CallToolResult, any, error) {
		e := &models.EndpointNode{
			ProjectID:    params.ProjectID,
			ServiceID:    params.ServiceID,
			URL:          params.URL,
			Method:       params.Method,
			Parameters:   params.Parameters,
			DiscoveredBy: params.DiscoveredBy,
			Status:       "discovered",
		}
		if err := s.CreateEndpoint(e); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(e)
		return mcputil.TextResult(string(b)), nil, nil
	})

	// endpoint_update_status
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "endpoint_update_status",
		Description: "Update an endpoint's testing status (tested|vulnerable|safe)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params endpointUpdateStatusParams) (*mcp.CallToolResult, any, error) {
		if err := s.UpdateEndpointStatus(params.ID, params.Status, params.TestedBy); err != nil {
			return mcputil.TextResult("update failed: " + err.Error()), nil, nil
		}
		return mcputil.TextResult("updated"), nil, nil
	})

	// find_untested_endpoints
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "find_untested_endpoints",
		Description: "Find all endpoints with status=discovered in a project",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID string `json:"project_id" jsonschema:"Project ID"`
	}) (*mcp.CallToolResult, any, error) {
		items, err := s.GetUntestedEndpoints(params.ProjectID)
		if err != nil {
			return mcputil.TextResult("query failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(items)
		return mcputil.TextResult(string(b)), nil, nil
	})
}

// ── Session tools ──

func registerSessionTools(server *mcp.Server, s store.Store, sm *mcputil.SessionMap) {
	// session_list
	Lister(server, "session_list", "List all session nodes by project ID", "Session", s.ListSessions)

	// session_get
	Getter(server, "session_get", "Get a session node by ID", "Session", s.GetSession)

	// session_create
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "session_create",
		Description: "Create a new session node",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params sessionParams) (*mcp.CallToolResult, any, error) {
		sess := &models.SessionNode{
			ProjectID:   params.ProjectID,
			SessionType: params.SessionType,
			CreatedBy:   params.CreatedBy,
			URL:         params.URL,
			AssetID:     params.AssetID,
		}
		if err := s.CreateSession(sess); err != nil {
			return mcputil.TextResult("create failed: " + err.Error()), nil, nil
		}
		b, _ := json.Marshal(sess)
		return mcputil.TextResult(string(b)), nil, nil
	})

	// find_sessions — filter by session_type
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "find_sessions",
		Description: "Find sessions by project ID, optionally filtered by session_type",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID   string `json:"project_id" jsonschema:"Project ID"`
		SessionType string `json:"session_type,omitempty" jsonschema:"Filter by session type (web/c2_shell/ssh/tunnel)"`
	}) (*mcp.CallToolResult, any, error) {
		items, err := s.ListSessions(params.ProjectID)
		if err != nil {
			return mcputil.TextResult("query failed: " + err.Error()), nil, nil
		}
		if params.SessionType == "" {
			b, _ := json.Marshal(items)
			return mcputil.TextResult(string(b)), nil, nil
		}
		var filtered []models.SessionNode
		for _, sess := range items {
			if sess.SessionType == params.SessionType {
				filtered = append(filtered, sess)
			}
		}
		b, _ := json.Marshal(filtered)
		return mcputil.TextResult(string(b)), nil, nil
	})
}
