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
func registerSchedulerBridge(server *mcp.Server, sm *mcputil.SessionMap) {
	// scheduler_create_task — forward to acasched
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "scheduler_create_task",
		Description: "Create a sub-task for another agent to execute (via acasched)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		ParentID    string `json:"parent_id" jsonschema:"Parent task ID"`
		Agent       string `json:"agent" jsonschema:"Target agent name"`
		Title       string `json:"title" jsonschema:"Task title"`
		Description string `json:"description,omitempty" jsonschema:"Task description"`
		MaxTurns    int    `json:"max_turns,omitempty" jsonschema:"Maximum turns for the sub-task"`
	}) (*mcp.CallToolResult, any, error) {
		body, _ := json.Marshal(map[string]any{
			"parent_id":   params.ParentID,
			"agent":       params.Agent,
			"title":       params.Title,
			"description": params.Description,
			"max_turns":   params.MaxTurns,
			"created_by":  "agent",
		})
		resp, err := http.Post(
			"http://127.0.0.1:9090/api/tasks",
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
		body, _ := json.Marshal(map[string]string{"result": params.Result})
		httpReq, err := http.NewRequestWithContext(
			ctx,
			"PATCH",
			"http://127.0.0.1:9090/api/tasks/"+params.TaskID,
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
