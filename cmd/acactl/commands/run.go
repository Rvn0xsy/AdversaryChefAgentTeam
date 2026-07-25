// cmd/acactl/commands/run.go
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func Run(acaPort int, goal, projectID string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	base := fmt.Sprintf("http://127.0.0.1:%d", acaPort)

	// Auto-create project if none provided
	if projectID == "" {
		body, err := json.Marshal(map[string]string{
			"name":        goal,
			"description": goal,
		})
		if err != nil {
			return fmt.Errorf("marshal project: %w", err)
		}
		resp, err := client.Post(base+"/api/projects", "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("acasched unreachable: %w", err)
		}
		if resp.StatusCode >= 300 {
			resp.Body.Close()
			return fmt.Errorf("acasched returned %d on project create", resp.StatusCode)
		}
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return fmt.Errorf("parse project response: %w", err)
		}
		resp.Body.Close()
		id, ok := result["id"].(string)
		if !ok {
			return fmt.Errorf("no project id in response")
		}
		projectID = id
		fmt.Printf("Project created: %s\n", projectID)
	}

	// Dispatch task
	taskBody, err := json.Marshal(map[string]string{
		"project_id":  projectID,
		"agent":       "supervisor",
		"title":       goal,
		"description": goal,
		"created_by":  "cli",
	})
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	resp, err := client.Post(base+"/api/tasks", "application/json", bytes.NewReader(taskBody))
	if err != nil {
		return fmt.Errorf("acasched unreachable: %w", err)
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return fmt.Errorf("acasched returned %d on task create", resp.StatusCode)
	}
	var taskResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&taskResult); err != nil {
		resp.Body.Close()
		return fmt.Errorf("parse task response: %w", err)
	}
	resp.Body.Close()

	taskID, ok := taskResult["id"].(string)
	if !ok {
		return fmt.Errorf("no task id in response")
	}
	fmt.Printf("Task %s dispatched. Use 'acactl logs %s' to observe progress.\n", taskID, taskID)

	// Poll until terminal
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := client.Get(base + "/api/tasks/" + taskID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "poll error: %v\n", err)
			continue
		}
		var task struct {
			Status string `json:"status"`
			Title  string `json:"title"`
		}
		json.NewDecoder(resp.Body).Decode(&task)
		resp.Body.Close()

		switch task.Status {
		case "done":
			fmt.Printf("Task completed: %s\n", task.Title)
			return nil
		case "failed":
			return fmt.Errorf("task %s failed", taskID)
		case "timeout":
			return fmt.Errorf("task %s timed out", taskID)
		}
		// still running, poll again
	}
	return nil
}
