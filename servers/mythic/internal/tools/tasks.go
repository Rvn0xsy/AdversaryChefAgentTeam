package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/mcputil"
	"adversarychef/mythic/internal/client"
)

func registerTasks(server *mcp.Server, c *client.Client) {
	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "mythic_issue_task",
		Description: "Issue a command to a callback. Returns the task ID. Use mythic_get_task_status or mythic_wait_for_task to get results.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		CallbackID string `json:"callback_id" jsonschema:"Callback display ID to task"`
		Command    string `json:"command" jsonschema:"Command to execute (e.g. shell, ls, ps, download)"`
		Parameters string `json:"parameters,omitempty" jsonschema:"Command parameters"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.IssueTask(ctx, &client.TaskAgentInput{
			CallbackID: params.CallbackID,
			Command:    params.Command,
			Parameters: params.Parameters,
		})
		if err != nil {
			return mcputil.TextResult("task failed: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "mythic_list_tasks",
		Description: "List tasks for a callback, most recent first",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		CallbackID string `json:"callback_id" jsonschema:"Callback display ID"`
		Limit      int    `json:"limit,omitempty" jsonschema:"Max tasks to return, default 50"`
	}) (*mcp.CallToolResult, any, error) {
		limit := params.Limit
		if limit <= 0 {
			limit = 50
		}
		tasks, err := c.ListTasksByCallback(ctx, params.CallbackID, limit)
		if err != nil {
			return mcputil.TextResult("failed to list tasks: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(tasks, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "mythic_get_task_status",
		Description: "Check the status of a task (non-blocking)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		TaskID string `json:"task_id" jsonschema:"Task display ID"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.GetTaskStatus(ctx, &client.GetTaskStatusInput{TaskID: params.TaskID})
		if err != nil {
			return mcputil.TextResult("failed to get task status: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "mythic_wait_for_task",
		Description: "Wait for a task to complete and return the output (blocking)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		TaskID  string `json:"task_id" jsonschema:"Task display ID"`
		Timeout int    `json:"timeout,omitempty" jsonschema:"Timeout in seconds, default 60"`
	}) (*mcp.CallToolResult, any, error) {
		timeout := params.Timeout
		if timeout <= 0 {
			timeout = 60
		}
		res, err := c.WaitForTask(ctx, &client.WaitForTaskInput{TaskID: params.TaskID, Timeout: timeout})
		if err != nil {
			return mcputil.TextResult("wait failed: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})

	mcputil.AddLoggingTool(server, &mcp.Tool{
		Name:        "mythic_get_task_output",
		Description: "Get the full decoded output of a completed task",
	}, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
		TaskID string `json:"task_id" jsonschema:"Task display ID"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := c.GetTaskOutput(ctx, &client.GetTaskOutputInput{TaskID: params.TaskID})
		if err != nil {
			return mcputil.TextResult("failed to get output: " + err.Error()), nil, nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return mcputil.TextResult(string(b)), nil, nil
	})
}
