// cmd/acactl/main.go
package main

import (
	"flag"
	"fmt"
	"os"

	"adversarychef/acactl/commands"
)

func main() {
	dataDir := flag.String("data-dir", "", "data directory (default: ~/.aca)")
	projectRoot := flag.String("project-root", ".", "project root for go build + prompts")
	var ports [3]int
	flag.IntVar(&ports[0], "nexus-port", 8081, "nexus-mcp port")
	flag.IntVar(&ports[1], "kali-port", 8080, "kali-mcp port")
	flag.IntVar(&ports[2], "acasched-port", 9090, "acasched port")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: acactl <command> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  up       Start all infrastructure\n")
		fmt.Fprintf(os.Stderr, "  down     Stop all services\n")
		fmt.Fprintf(os.Stderr, "  status   Check service health\n")
		fmt.Fprintf(os.Stderr, "  tasks    List tasks\n")
		fmt.Fprintf(os.Stderr, "  logs     View task execution logs\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	// Parse subcommand
	subcmd := os.Args[1]
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	flag.Parse()

	switch subcmd {
	case "up":
		if err := commands.Up(*dataDir, *projectRoot, ports); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "down":
		if err := commands.Down(*dataDir, ports); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := commands.Status(ports); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", subcmd)
		flag.Usage()
		os.Exit(1)
	}
}
