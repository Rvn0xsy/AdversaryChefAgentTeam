// cmd/acactl/main.go
package main

import (
	"flag"
	"fmt"
	"os"

	"adversarychef/acactl/commands"
)

// commands that only use global flags (no subcommand-specific flags)
var globalCommands = map[string]bool{
	"up":     true,
	"down":   true,
	"status": true,
}

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
		fmt.Fprintf(os.Stderr, "  run      Dispatch a task and wait for completion\n")
		fmt.Fprintf(os.Stderr, "  tasks    List tasks\n")
		fmt.Fprintf(os.Stderr, "  logs     View task execution logs\n")
		fmt.Fprintf(os.Stderr, "  project  Create a project\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	// Extract subcommand
	subcmd := os.Args[1]

	// Parse global flags only for commands that use them
	if globalCommands[subcmd] {
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		flag.Parse()
	}

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
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		goal := fs.String("goal", "", "Task goal (required)")
		project := fs.String("project", "", "Project ID (auto-created if empty)")
		fs.Parse(os.Args[2:])
		if *goal == "" {
			fs.Usage()
			os.Exit(1)
		}
		if err := commands.Run(ports[2], *goal, *project); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "tasks":
		fs := flag.NewFlagSet("tasks", flag.ExitOnError)
		project := fs.String("project", "", "filter by project ID")
		st := fs.String("status", "", "filter by status")
		fs.Parse(os.Args[2:])
		if err := commands.Tasks(ports[2], *project, *st); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "logs":
		fs := flag.NewFlagSet("logs", flag.ExitOnError)
		taskID := fs.String("task", "", "task ID")
		follow := fs.Bool("follow", false, "follow streaming output")
		raw := fs.Bool("raw", false, "output raw stream-json")
		fs.Parse(os.Args[2:])
		if err := commands.Logs(ports[2], *taskID, *follow, *raw); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "project":
		fs := flag.NewFlagSet("project", flag.ExitOnError)
		name := fs.String("name", "", "project name")
		desc := fs.String("description", "", "project description")
		fs.Parse(os.Args[2:])
		if err := commands.CreateProject(ports[2], *name, *desc); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", subcmd)
		flag.Usage()
		os.Exit(1)
	}
}
