// cmd/acactl/commands/project.go
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type apiProject struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func ProjectList(acaPort int) error {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/projects", acaPort))
	if err != nil {
		return fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()

	var projects []apiProject
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return fmt.Errorf("parse projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	fmt.Println("┌──────────────────────────────────────┬─────────┬────────────┬──────────────────────────────────────────────┐")
	fmt.Println("│ PROJECT ID                           │ STATUS  │ CREATED    │ NAME / DESCRIPTION                           │")
	fmt.Println("├──────────────────────────────────────┼─────────┼────────────┼──────────────────────────────────────────────┤")
	for _, p := range projects {
		desc := p.Description
		if len(desc) > 44 {
			desc = desc[:41] + "..."
		}
		status := p.Status
		switch status {
		case "active":
			status = "\033[32mactive\033[0m"
		case "stopped":
			status = "\033[33mstopped\033[0m"
		}
		fmt.Printf("│ %-36s │ %-24s │ %-10s │ %-44s │\n", p.ID, status, p.CreatedAt[:10], p.Name)
		fmt.Printf("│ %-36s │ %-24s │ %-10s │ %-44s │\n", "", "", "", desc)
	}
	fmt.Println("└──────────────────────────────────────┴─────────┴────────────┴──────────────────────────────────────────────┘")
	return nil
}

func ProjectCreate(acaPort int, name, description string) error {
	body, _ := json.Marshal(map[string]string{"name": name, "description": description})
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/api/projects", acaPort),
		"application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()

	var p apiProject
	json.NewDecoder(resp.Body).Decode(&p)
	fmt.Printf("Project created: %s (%s)\n", p.ID, p.Name)
	return nil
}

func ProjectStop(acaPort int, name string) error {
	return setProjectStatus(acaPort, name, "stopped")
}

func ProjectResume(acaPort int, name string) error {
	return setProjectStatus(acaPort, name, "active")
}

func setProjectStatus(acaPort int, name, status string) error {
	id, err := findProjectByName(acaPort, name)
	if err != nil {
		return err
	}

	body, _ := json.Marshal(map[string]string{"status": status})
	req, _ := http.NewRequest("PATCH",
		fmt.Sprintf("http://127.0.0.1:%d/api/projects/%s", acaPort, id),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Project %s → %s\n", name, status)
	return nil
}

func findProjectByName(acaPort int, name string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/projects", acaPort))
	if err != nil {
		return "", fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()

	var projects []apiProject
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return "", fmt.Errorf("parse projects: %w", err)
	}

	for _, p := range projects {
		if p.Name == name {
			return p.ID, nil
		}
	}
	return "", fmt.Errorf("project '%s' not found", name)
}
