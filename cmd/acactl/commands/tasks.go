// cmd/acactl/commands/tasks.go
package commands

import (
	"encoding/json"
	"fmt"
	"net/http"

	"adversarychef/acactl/display"
)

func Tasks(acaPort int, projectID, status string) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/tasks", acaPort)
	if projectID != "" {
		url += "?project_id=" + projectID
	}
	if status != "" {
		url += "&status=" + status
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()

	var tasks []struct {
		ID, Agent, Status, Title string
	}
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return fmt.Errorf("parse tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	var summaries []display.TaskSummary
	for _, t := range tasks {
		summaries = append(summaries, display.TaskSummary{
			ID: t.ID, Agent: t.Agent, Status: t.Status, Title: t.Title,
		})
	}
	display.PrintTaskTable(summaries)
	return nil
}
