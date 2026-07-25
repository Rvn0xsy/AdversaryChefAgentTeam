// cmd/acactl/commands/projects.go
package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func CreateProject(acaPort int, name, description string) error {
	body, _ := json.Marshal(map[string]string{"name": name, "description": description})
	resp, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/api/projects", acaPort),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("acasched unreachable: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("Project created: %s\n", result["id"])
	return nil
}
