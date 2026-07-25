// cmd/acactl/main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"adversarychef/acactl/commands"
)

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

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	subcmd := os.Args[1]

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
		goal := fs.String("goal", "", "Task goal (required unless --project)")
		project := fs.String("project", "", "Existing project name or ID")
		detach := fs.Bool("detach", false, "Run in background, don't stream logs")
		fs.Parse(os.Args[2:])
		if *goal == "" && *project == "" {
			fmt.Fprintln(os.Stderr, "Usage: acactl run -goal <text> [-project <name>] [--detach]")
			os.Exit(1)
		}
		if err := commands.Run(ports[2], *goal, *project, *detach); err != nil {
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
		follow := fs.Bool("follow", false, "follow streaming output")
		raw := fs.Bool("raw", false, "output raw stream-json")
		fs.Parse(os.Args[2:])
		arg := fs.Arg(0)
		if arg == "" {
			fmt.Fprintln(os.Stderr, "Usage: acactl logs <task-id|project-name-or-id> [--follow] [--raw]")
			os.Exit(1)
		}
		// Determine if arg is a task ID or project name
		if strings.HasPrefix(arg, "task_") || strings.HasPrefix(arg, "proj_") {
			if strings.HasPrefix(arg, "proj_") {
				// Project ID: show all tasks in project
				if err := commands.ProjectLogs(ports[2], arg); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Task ID
				if err := commands.Logs(ports[2], arg, *follow, *raw); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
			}
		} else {
			// Try as project name first, then as task
			if err := commands.ProjectLogs(ports[2], arg); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}
	case "project":
		sub := ""
		var projName, projDesc string
		if len(os.Args) > 2 {
			sub = os.Args[2]
		}
		// Manual flag parsing for project subcommands
		for i := 3; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "-name":
				if i+1 < len(os.Args) {
					projName = os.Args[i+1]
					i++
				}
			case "-description":
				if i+1 < len(os.Args) {
					projDesc = os.Args[i+1]
					i++
				}
			}
		}
		switch sub {
		case "list", "":
			if err := commands.ProjectList(ports[2]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		case "create":
			if projName == "" {
				fmt.Fprintln(os.Stderr, "Usage: acactl project create -name <name> -description <desc>")
				os.Exit(1)
			}
			if err := commands.ProjectCreate(ports[2], projName, projDesc); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		case "stop":
			if projName == "" {
				fmt.Fprintln(os.Stderr, "Usage: acactl project stop -name <name>")
				os.Exit(1)
			}
			if err := commands.ProjectStop(ports[2], projName); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		case "resume":
			if projName == "" {
				fmt.Fprintln(os.Stderr, "Usage: acactl project resume -name <name>")
				os.Exit(1)
			}
			if err := commands.ProjectResume(ports[2], projName); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Usage: acactl project [list|create|stop|resume]\n")
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", subcmd)
		os.Exit(1)
	}
}