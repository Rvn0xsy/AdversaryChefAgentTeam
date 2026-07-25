// cmd/acactl/commands/up.go
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"adversarychef/acactl/lifecycle"
)

func Up(dataDir, projectRoot string, ports [3]int) error {
	home, _ := os.UserHomeDir()
	if dataDir == "" {
		dataDir = home + "/.aca"
	}

	pm := &lifecycle.ProcessManager{
		BinDir:  dataDir + "/bin",
		DataDir: dataDir + "/data",
		LogDir:  dataDir + "/logs",
		PidDir:  dataDir + "/pids",
	}
	os.MkdirAll(pm.BinDir, 0755)
	os.MkdirAll(pm.DataDir, 0755)
	os.MkdirAll(pm.LogDir, 0755)
	os.MkdirAll(pm.PidDir, 0755)

	// Check ports
	for _, p := range ports {
		if err := lifecycle.CheckPort(p); err != nil {
			return fmt.Errorf("port conflict: %w", err)
		}
	}

	nexusPort, kaliPort, acaPort := ports[0], ports[1], ports[2]
	nexusDB := filepath.Join(pm.DataDir, "nexus.db")
	acaschedDB := filepath.Join(pm.DataDir, "acasched.db")
	logDir := filepath.Join(pm.LogDir, "tasks")

	// Build binaries
	fmt.Println("Building nexus-mcp...")
	if err := pm.BuildBinary("./servers/nexus/cmd/server", filepath.Join(pm.BinDir, "nexus-mcp")); err != nil {
		return fmt.Errorf("build nexus-mcp: %w", err)
	}
	fmt.Println("Building acasched...")
	if err := pm.BuildBinary("./cmd/acasched", filepath.Join(pm.BinDir, "acasched")); err != nil {
		return fmt.Errorf("build acasched: %w", err)
	}

	// Check kali-mcp image
	fmt.Println("Checking kali-mcp image...")
	// podman image exists kali-mcp

	// Start nexus-mcp
	fmt.Println("Starting nexus-mcp...")
	nexus := &lifecycle.Service{
		Name:   "nexus-mcp",
		Port:   nexusPort,
		Binary: filepath.Join(pm.BinDir, "nexus-mcp"),
		Args:   []string{"-db", nexusDB},
	}
	if err := pm.Start(nexus); err != nil {
		return fmt.Errorf("start nexus-mcp: %w", err)
	}
	if err := pm.HealthCheck(nexus); err != nil {
		return fmt.Errorf("health nexus-mcp: %w", err)
	}

	// Start kali-mcp
	fmt.Println("Starting kali-mcp...")
	kali := &lifecycle.Service{
		Name:   "kali-mcp",
		Port:   kaliPort,
		Binary: "kali-mcp",
		Podman: true,
	}
	if err := pm.Start(kali); err != nil {
		return fmt.Errorf("start kali-mcp: %w", err)
	}
	if err := pm.HealthCheck(kali); err != nil {
		return fmt.Errorf("health kali-mcp: %w", err)
	}

	// Start acasched
	fmt.Println("Starting acasched...")
	acasched := &lifecycle.Service{
		Name:   "acasched",
		Port:   acaPort,
		Binary: filepath.Join(pm.BinDir, "acasched"),
		Args: []string{
			"-db", acaschedDB,
			"-nexus-mcp", fmt.Sprintf("http://127.0.0.1:%d", nexusPort),
			"-kali-mcp", fmt.Sprintf("http://127.0.0.1:%d", kaliPort),
			"-prompts", filepath.Join(projectRoot, "prompts"),
			"-log-dir", logDir,
		},
	}
	if err := pm.Start(acasched); err != nil {
		return fmt.Errorf("start acasched: %w", err)
	}
	if err := pm.HealthCheck(acasched); err != nil {
		return fmt.Errorf("health acasched: %w", err)
	}

	// Output status
	fmt.Println()
	fmt.Println("┌────────────┬─────────┬───────┐")
	fmt.Println("│ SERVICE    │ STATUS  │ PORT  │")
	fmt.Println("├────────────┼─────────┼───────┤")
	fmt.Printf("│ nexus-mcp  │ running │ %-5d │\n", nexusPort)
	fmt.Printf("│ kali-mcp   │ running │ %-5d │\n", kaliPort)
	fmt.Printf("│ acasched   │ running │ %-5d │\n", acaPort)
	fmt.Println("└────────────┴─────────┴───────┘")
	fmt.Println()
	fmt.Println("All services started.")
	return nil
}
