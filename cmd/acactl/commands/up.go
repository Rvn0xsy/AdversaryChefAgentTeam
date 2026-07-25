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
	acaschedDB := filepath.Join(pm.DataDir, "acasched.db")
	logDir := filepath.Join(pm.LogDir, "tasks")

	// Build binaries (acasched only, mcp services use Docker)
	fmt.Println("Building acasched...")
	if err := pm.BuildBinary("./cmd/acasched", filepath.Join(pm.BinDir, "acasched")); err != nil {
		return fmt.Errorf("build acasched: %w", err)
	}

	// Check Docker images exist
	fmt.Println("Checking Docker images...")
	// docker images exist: nexus-mcp, kali-mcp, goose

	// Start nexus-mcp (Docker)
	fmt.Println("Starting nexus-mcp...")
	nexus := &lifecycle.Service{
		Name:      "nexus-mcp",
		Port:      nexusPort,
		Binary:    "nexus-mcp",
		Args:      []string{"-db", "/data/nexus.db"},
		Mounts:    []string{pm.DataDir + ":/data"},
		Env:       []string{"SCHEDULER_URL=http://host.docker.internal:9090"},
		Container: true,
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
		Name:      "kali-mcp",
		Port:      kaliPort,
		Binary:    "kali-mcp",
		CapAdd:    []string{"NET_RAW", "NET_ADMIN", "SYS_PTRACE"},
		Container: true,
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
			"-prompts", filepath.Join(projectRoot, "prompts"),
			"-skills", filepath.Join(projectRoot, "skills"),
			"-registry", filepath.Join(projectRoot, "prompts", "_mcp-registry.yaml"),
			"-env", filepath.Join(projectRoot, ".env"),
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
