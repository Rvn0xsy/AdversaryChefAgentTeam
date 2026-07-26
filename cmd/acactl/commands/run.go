// cmd/acactl/commands/run.go
package commands

import (
	"adversarychef/acactl/display"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func Run(acaPort int, goal, projectName string, detach bool) error {
	client := &http.Client{Timeout: 30 * time.Second}
	base := fmt.Sprintf("http://127.0.0.1:%d", acaPort)

	// Resolve project: use existing or create new
	projectID, err := resolveProject(client, base, goal, projectName)
	if err != nil {
		return err
	}

	// Dispatch initial supervisor task
	taskID, err := dispatchTask(client, base, projectID, "red-team/supervisor", goal, goal)
	if err != nil {
		return err
	}

	if detach {
		fmt.Printf("Task %s dispatched (detached). Use 'acactl logs %s' to observe progress.\n", taskID, taskID)
		return nil
	}

	// Sequential task following — chase the project chain
	return followProject(client, base, projectID, taskID)
}

// resolveProject finds existing project by name or creates a new one.
func resolveProject(client *http.Client, base, goal, projectName string) (string, error) {
	if projectName != "" {
		id, err := findProjectByNameURL(client, base, projectName)
		if err != nil {
			return "", err
		}
		fmt.Printf("Project: %s (%s)\n", projectName, id)
		return id, nil
	}
	// Auto-create
	body, _ := json.Marshal(map[string]string{"name": goal, "description": goal})
	resp, err := client.Post(base+"/api/projects", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()
	var p struct{ ID string `json:"id"` }
	json.NewDecoder(resp.Body).Decode(&p)
	fmt.Printf("Project created: %s\n", p.ID)
	return p.ID, nil
}

func findProjectByNameURL(client *http.Client, base, name string) (string, error) {
	resp, err := client.Get(base + "/api/projects")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(resp.Body).Decode(&projects)
	for _, p := range projects {
		if p.Name == name {
			return p.ID, nil
		}
	}
	return "", fmt.Errorf("project '%s' not found", name)
}

func dispatchTask(client *http.Client, base, projectID, agent, title, desc string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"project_id":  projectID,
		"agent":       agent,
		"title":       title,
		"description": desc,
		"created_by":  "cli",
	})
	// Also set max_turns in the request body
	var bodyWithTurns map[string]interface{}
	json.Unmarshal(body, &bodyWithTurns) // lazy: parse the string map
	bodyWithTurns["max_turns"] = 150
	bodyWithTurns["timeout_secs"] = 7200
	body, _ = json.Marshal(bodyWithTurns)
	resp, err := client.Post(base+"/api/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()
	var t struct{ ID string `json:"id"` }
	json.NewDecoder(resp.Body).Decode(&t)
	return t.ID, nil
}

// followProject chases tasks in the project: stream one, wait for children, repeat.
func followProject(client *http.Client, base, projectID, firstTaskID string) error {
	taskID := firstTaskID
	followed := map[string]bool{}

	for taskID != "" {
		followed[taskID] = true

		// Wait for task to leave "pending"
		if err := waitForStart(client, base, taskID); err != nil {
			return err
		}

		// Stream its logs in real-time
		if err := streamTask(client, base, taskID); err != nil {
			return err
		}

		// Check status — if failed/timeout, stop following
		status := getTaskStatus(client, base, taskID)
		if status == "failed" || status == "timeout" {
			fmt.Printf("  \033[31m✗ %s\033[0m\n\n", status)
			break
		}

		fmt.Printf("  \033[32m✅ Completed\033[0m\n\n")

		// Find next task to follow (new child we haven't followed yet)
		taskID = findUnfollowedTask(client, base, projectID, followed)
	}

	// Final summary
	return printProjectSummary(client, base, projectID)
}

func findUnfollowedTask(client *http.Client, base, projectID string, followed map[string]bool) string {
	resp, err := client.Get(base + "/api/projects/" + projectID + "/tasks")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var tasks []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&tasks)

	for _, t := range tasks {
		if !followed[t.ID] && (t.Status == "running" || t.Status == "dispatched" || t.Status == "pending") {
			return t.ID
		}
	}
	return ""
}

func waitForStart(client *http.Client, base, taskID string) error {
	fmt.Printf("Task %s dispatched. Waiting for agent...\n", taskID)
	for {
		time.Sleep(500 * time.Millisecond)
		status := getTaskStatus(client, base, taskID)
		if status == "" {
			continue
		}
		if status != "pending" && status != "dispatched" {
			return nil
		}
	}
}

func streamTask(client *http.Client, base, taskID string) error {
	// Get agent name
	agent := getTaskAgent(client, base, taskID)

	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("  Task %s  [%s]\n\n", taskID, agent)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sseURL := base + "/api/tasks/" + taskID + "/logs?follow=true"
	req, _ := http.NewRequestWithContext(ctx, "GET", sseURL, nil)
	sseResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect to log stream: %w", err)
	}
	defer sseResp.Body.Close()

	sseDone := make(chan struct{})

	go func() {
		defer close(sseDone)
		scanner := bufio.NewScanner(sseResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")

			var msg struct {
				Type    string          `json:"type"`
				Message *struct {
					Role    string          `json:"role"`
					Content json.RawMessage `json:"content"`
				} `json:"message,omitempty"`
			}
			if err := json.Unmarshal([]byte(payload), &msg); err != nil {
				continue
			}

			if msg.Type == "message" && msg.Message != nil && msg.Message.Role == "assistant" {
				lines := display.FormatMessage([]byte(payload))
				for _, l := range lines {
					fmt.Println(l)
				}
			}
		}
	}()

	// Poll status until terminal
	time.Sleep(3 * time.Second)
	for {
		time.Sleep(2 * time.Second)
		status := getTaskStatus(client, base, taskID)
		switch status {
		case "done", "failed", "timeout":
			cancel()
			<-sseDone
			return nil
		}
	}
}

func printProjectSummary(client *http.Client, base, projectID string) error {
	resp, err := client.Get(base + "/api/projects/" + projectID + "/tasks")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var tasks []struct {
		ID     string `json:"id"`
		Agent  string `json:"agent"`
		Status string `json:"status"`
		Title  string `json:"title"`
	}
	json.NewDecoder(resp.Body).Decode(&tasks)

	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("  \033[1mProject Summary\033[0m\n\n")

	statusIcon := map[string]string{
		"done": "✅", "running": "🔄", "pending": "⏳",
		"failed": "❌", "timeout": "⏰", "dispatched": "📤",
	}

	for _, t := range tasks {
		icon := statusIcon[t.Status]
		if icon == "" {
			icon = "  "
		}
		agent := strings.TrimPrefix(t.Agent, "red-team/")
		fmt.Printf("  %s \033[36m%-15s\033[0m %s\n", icon, agent, t.Title)
	}

	if len(tasks) == 0 {
		fmt.Println("  No tasks found.")
	}
	fmt.Println()
	return nil
}

func getTaskStatus(client *http.Client, base, taskID string) string {
	resp, err := client.Get(base + "/api/tasks/" + taskID)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var t struct{ Status string `json:"status"` }
	json.NewDecoder(resp.Body).Decode(&t)
	return t.Status
}

func getTaskAgent(client *http.Client, base, taskID string) string {
	resp, err := client.Get(base + "/api/tasks/" + taskID)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var t struct{ Agent string `json:"agent"` }
	json.NewDecoder(resp.Body).Decode(&t)
	return strings.TrimPrefix(t.Agent, "red-team/")
}
