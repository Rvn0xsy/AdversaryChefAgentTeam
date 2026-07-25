// cmd/acactl/commands/logs.go
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"adversarychef/acactl/display"
)

func Logs(acaPort int, taskID string, follow, raw bool) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/tasks/%s/logs", acaPort, taskID)
	if follow {
		url += "?follow=true"
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		fmt.Println("Logs not found for task:", taskID)
		return nil
	}

	if raw {
		io.Copy(os.Stdout, resp.Body)
		return nil
	}

	// Fetch task to get agent name
	base := fmt.Sprintf("http://127.0.0.1:%d", acaPort)
	var agent string
	if tr, err := http.Get(base + "/api/tasks/" + taskID); err == nil {
		var t struct{ Agent string `json:"agent"` }
		json.NewDecoder(tr.Body).Decode(&t)
		tr.Body.Close()
		agent = strings.TrimPrefix(t.Agent, "red-team/")
	}

	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("  Task %s  [%s]\n\n", taskID, agent)
	return display.FormatStreamJSON(resp.Body)
}

// ProjectLogs shows all tasks in a project (by name or ID).
func ProjectLogs(acaPort int, nameOrID string) error {
	base := fmt.Sprintf("http://127.0.0.1:%d", acaPort)

	// If not a proj_ ID, resolve name
	projectID := nameOrID
	if !strings.HasPrefix(nameOrID, "proj_") {
		id, err := findProjectByNameURL(&http.Client{Timeout: 10 * 1000000000}, base, nameOrID)
		if err != nil {
			return err
		}
		projectID = id
	}

	resp, err := http.Get(base + "/api/projects/" + projectID + "/tasks")
	if err != nil {
		return fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()

	var tasks []struct {
		ID        string `json:"id"`
		ParentID  string `json:"parent_id"`
		Agent     string `json:"agent"`
		Status    string `json:"status"`
		Title     string `json:"title"`
	}

	statusIcon := map[string]string{
		"done": "✅", "running": "🔄", "pending": "⏳",
		"failed": "❌", "timeout": "⏰", "dispatched": "📤",
	}

	json.NewDecoder(resp.Body).Decode(&tasks)
	if len(tasks) == 0 {
		fmt.Println("No tasks in project.")
		return nil
	}

	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("  Project %s\n\n", projectID)
	fmt.Println("┌──────────────────────┬──────────┬─────────┬─────────────────────────────────────────┐")
	fmt.Println("│ TASK ID              │ AGENT    │ STATUS  │ TITLE                                   │")
	fmt.Println("├──────────────────────┼──────────┼─────────┼─────────────────────────────────────────┤")
	for _, t := range tasks {
		icon := statusIcon[t.Status]
		agent := strings.TrimPrefix(t.Agent, "red-team/")
		title := t.Title
		if len(title) > 39 {
			title = title[:36] + "..."
		}
		fmt.Printf("│ %-20s │ %-8s │ %s %-5s │ %-39s │\n",
			truncate(t.ID, 20), truncate(agent, 8), icon, t.Status, title)
	}
	fmt.Println("└──────────────────────┴──────────┴─────────┴─────────────────────────────────────────┘")
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
