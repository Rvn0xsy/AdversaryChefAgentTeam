// cmd/acactl/commands/logs.go
package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"

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

	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("  Task %s\n\n", taskID)
	return display.FormatStreamJSON(resp.Body)
}
