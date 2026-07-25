package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/mcputil"
)

// registerSchedulerBridge registers tools for the scheduler integration bridge.
func registerSchedulerBridge(server *mcp.Server, sm *mcputil.SessionMap, schedulerURL string) {
	// scheduler_create_task — forward to acasched
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "scheduler_create_task",
		Description: "Create a sub-task for another agent to execute (via acasched)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ProjectID   string `json:"project_id,omitempty" jsonschema:"Project ID (auto-injected from session)"`
		ParentID    string `json:"parent_id,omitempty" jsonschema:"Parent task ID (optional)"`
		Agent       string `json:"agent" jsonschema:"Target agent name"`
		Title       string `json:"title" jsonschema:"Task title"`
		Description string `json:"description,omitempty" jsonschema:"Task description"`
		MaxTurns    int    `json:"max_turns,omitempty" jsonschema:"Maximum turns for the sub-task (minimum 30, default 40)"`
	}) (*mcp.CallToolResult, any, error) {
		projectID := mcputil.ProjectIDFromContext(ctx)
		if projectID == "" { projectID = params.ProjectID }
		if projectID == "" {
			return mcputil.TextResult("project_id not found in session context"), nil, nil
		}
		// Enforce minimum max_turns for long-running tasks
		maxTurns := params.MaxTurns
		if maxTurns < 150 {
			maxTurns = 150
		}
		body, _ := json.Marshal(map[string]any{
			"parent_id":   params.ParentID,
			"project_id":  projectID,
			"agent":       params.Agent,
			"title":       params.Title,
			"description": params.Description,
			"max_turns":   maxTurns,
			"created_by":  "agent",
		})
		resp, err := http.Post(
			schedulerURL+"/api/tasks",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			return mcputil.TextResult("scheduler unreachable: " + err.Error()), nil, nil
		}
		defer resp.Body.Close()

		var task map[string]any
		json.NewDecoder(resp.Body).Decode(&task)
		b, _ := json.Marshal(task)
		return mcputil.TextResult(string(b)), nil, nil
	})

	// scheduler_complete_task — mark own task as complete
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "scheduler_complete_task",
		Description: "Mark your own task as complete",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		TaskID string `json:"task_id" jsonschema:"Task ID to complete"`
		Result string `json:"result,omitempty" jsonschema:"Result summary"`
	}) (*mcp.CallToolResult, any, error) {
		body, _ := json.Marshal(map[string]string{
			"status": "done",
			"result": params.Result,
		})
		httpReq, err := http.NewRequestWithContext(
			ctx,
			"PATCH",
			schedulerURL+"/api/tasks/"+params.TaskID,
			bytes.NewReader(body),
		)
		if err != nil {
			return mcputil.TextResult("request creation failed: " + err.Error()), nil, nil
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return mcputil.TextResult("scheduler unreachable: " + err.Error()), nil, nil
		}
		defer resp.Body.Close()

		return mcputil.TextResult("task marked done"), nil, nil
	})
}

