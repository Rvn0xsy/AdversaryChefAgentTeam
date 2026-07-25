// cmd/acactl/commands/down.go
package commands

import (
	"fmt"
	"os"

	"adversarychef/acactl/lifecycle"
)

func Down(dataDir string, ports [3]int) error {
	home, _ := os.UserHomeDir()
	if dataDir == "" {
		dataDir = home + "/.aca"
	}

	pm := &lifecycle.ProcessManager{
		PidDir: dataDir + "/pids",
	}

	services := []*lifecycle.Service{
		{Name: "acasched", Port: ports[2]},
		{Name: "kali-mcp", Port: ports[1], Podman: true},
		{Name: "nexus-mcp", Port: ports[0]},
	}

	for _, svc := range services {
		fmt.Printf("Stopping %s...\n", svc.Name)
		if err := pm.Stop(svc); err != nil {
			fmt.Printf("  warning: %v\n", err)
		}
	}

	fmt.Println("All services stopped.")
	return nil
}
