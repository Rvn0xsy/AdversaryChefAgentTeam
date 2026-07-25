// cmd/acactl/commands/status.go
package commands

import (
	"fmt"
	"net/http"
)

func Status(ports [3]int) error {
	fmt.Println("┌────────────┬─────────┬───────┐")
	fmt.Println("│ SERVICE    │ STATUS  │ PORT  │")
	fmt.Println("├────────────┼─────────┼───────┤")
	check("nexus-mcp", ports[0])
	check("kali-mcp", ports[1])
	check("acasched", ports[2])
	fmt.Println("└────────────┴─────────┴───────┘")
	return nil
}

func check(name string, port int) {
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	resp, err := http.Get(url)
	status := "stopped"
	if err == nil && resp.StatusCode == 200 {
		status = "running"
		resp.Body.Close()
	} else if err == nil {
		status = "unhealthy"
		resp.Body.Close()
	}
	fmt.Printf("│ %-10s │ %-7s │ %-5d │\n", name, status, port)
}
