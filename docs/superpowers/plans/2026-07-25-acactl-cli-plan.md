# acactl CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `acactl` — a Go CLI binary for one-click infrastructure management and Agent execution observability.

**Architecture:** New Go module `cmd/acactl` with two internal packages (`lifecycle`, `display`). Modifies acasched's goose runner (log persistence) and API (new `/logs` endpoint). All components communicate via HTTP health checks and REST API calls.

**Tech Stack:** Go 1.26, stdlib `net/http`, `os/exec`, `encoding/json`, `text/tabwriter`, existing modules (acasched, nexus-mcp, kali-mcp via podman).

## Global Constraints

- Module paths: `adversarychef/acactl` (new), `adversarychef/acasched` (modified)
- Port defaults: nexus=8081, kali=8080, acasched=9090
- Data directory: `~/.aca/` (expand with `os.UserHomeDir`)
- Binary output: `~/.aca/bin/nexus-mcp`, `~/.aca/bin/acasched`
- PID files: `~/.aca/pids/` (PID on first line, no newline)
- Log files: `~/.aca/logs/tasks/<task_id>.jsonl` (one JSON object per line, goose stream-json format)
- Build: `GOWORK=off CGO_ENABLED=0 go build`
- podman: `podman run -d --name kali-mcp --cap-add NET_RAW --cap-add NET_ADMIN --sysctl net.ipv6.conf.all.disable_ipv6=0 -p 8080:8080 kali-mcp`
- acasched new flag: `-log-dir string` default `~/.aca/logs/tasks/`
- STREAM-JSON format: `{"type":"tool_call"|"tool_result"|"assistant",...}` (goose `--output-format stream-json`)

---

## File Structure

```
Create: cmd/acactl/go.mod
Create: cmd/acactl/main.go
Create: cmd/acactl/lifecycle/manager.go
Create: cmd/acactl/lifecycle/ports.go
Create: cmd/acactl/commands/up.go
Create: cmd/acactl/commands/down.go
Create: cmd/acactl/commands/status.go
Create: cmd/acactl/commands/tasks.go
Create: cmd/acactl/commands/logs.go
Create: cmd/acactl/commands/projects.go
Create: cmd/acactl/display/table.go
Create: cmd/acactl/display/stream.go

Modify: cmd/acasched/main.go (add -log-dir flag, pass to Runner + API)
Modify: cmd/acasched/internal/goose/runner.go (add LogDir field, write stdout to file)
Modify: cmd/acasched/internal/api/server.go (register /api/tasks/{id}/logs route)
Modify: cmd/acasched/internal/api/logs.go (new file: handleTaskLogs + SSE)
Modify: go.work (add ./cmd/acactl)
```

---

### Task 1: acasched log persistence + logs API endpoint

**Files:**
- Modify: `cmd/acasched/main.go`
- Modify: `cmd/acasched/internal/goose/runner.go`
- Create: `cmd/acasched/internal/api/logs.go`
- Modify: `cmd/acasched/internal/api/server.go`

**Interfaces:**
- Produces: `Runner.LogDir string`, `Runner` writes goose stdout to `{LogDir}/{Task.ID}.jsonl`
- Produces: `GET /api/tasks/{id}/logs` endpoint, `GET /api/tasks/{id}/logs?follow=true` SSE stream

- [ ] **Step 1: Add LogDir to Runner and write stdout to file**

```go
// cmd/acasched/internal/goose/runner.go — modify Runner struct and Execute()

type Runner struct {
	PromptsDir string
	WorkDir    string
	LogDir     string    // NEW
	NexusMCP   string
	KaliMCP    string
	MythicMCP  string
}
```

In `Execute()`, after building the command but before `cmd.Run()`, add:

```go
// NEW: log persistence
if r.LogDir != "" {
	os.MkdirAll(r.LogDir, 0755)
	logPath := filepath.Join(r.LogDir, task.ID+".jsonl")
	f, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	defer f.Close()
	
	// Copy stderr to same file (prefixed)
	cmd.Stderr = f
	
	// Pipe stdout: write to file + capture for parsing
	pr, pw, _ := os.Pipe()
	cmd.Stdout = pw
	
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.TeeReader(pr, io.MultiWriter(f, &buf))
		pw.Close()
		close(done)
	}()
	
	err = cmd.Run()
	<-done
	if err != nil {
		return nil, fmt.Errorf("goose exited: %w", err)
	}
	return parseStreamOutput(buf.String()), nil
}

// Fallback for when LogDir is empty (existing behavior)
output, err := cmd.CombinedOutput()
```

Note: add `"bytes"`, `"io"`, `"path/filepath"` to imports.

- [ ] **Step 2: Add -log-dir flag to acasched main.go**

```go
// cmd/acasched/main.go — add after existing flag declarations
logDir := flag.String("log-dir", "", "task log directory (default: ~/.aca/logs/tasks/)")

// In Runner initialization:
runner := &goose.Runner{
	PromptsDir: *promptsDir,
	LogDir:     *logDir,   // NEW
	NexusMCP:   *nexusURL,
	KaliMCP:    *kaliURL,
	MythicMCP:  *mythicURL,
}
```

- [ ] **Step 3: Create logs API handler**

```go
// cmd/acasched/internal/api/logs.go (new file)
package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func handleTaskLogs(logDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := r.PathValue("id")
		if taskID == "" {
			http.Error(w, "missing task id", http.StatusBadRequest)
			return
		}
		logPath := filepath.Join(logDir, taskID+".jsonl")

		follow := r.URL.Query().Get("follow") == "true"
		if follow {
			serveSSELog(w, r, logPath)
			return
		}

		data, err := os.ReadFile(logPath)
		if err != nil {
			http.Error(w, "log not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Write(data)
	}
}

func serveSSELog(w http.ResponseWriter, r *http.Request, logPath string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	f, err := os.Open(logPath)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
		flusher.Flush()
	}
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}
```

- [ ] **Step 4: Register route in server.go**

```go
// cmd/acasched/internal/api/server.go — add after existing routes
mux.HandleFunc("GET /api/tasks/{id}/logs", handleTaskLogs(logDir))
```

Update `RunAPI` signature to accept `logDir string`:

```go
func RunAPI(ctx context.Context, s *store.Store, logDir string, port int) {
    // ...
    mux.HandleFunc("GET /api/tasks/{id}/logs", handleTaskLogs(logDir))
```

Update call in main.go: `go api.RunAPI(ctx, s, *logDir, *port)`

- [ ] **Step 5: Build and verify**

```bash
go build ./cmd/acasched/...
```
Expected: clean build, no errors.

- [ ] **Step 6: Commit**

```bash
git add cmd/acasched/
git commit -m "feat(acasched): log persistence for goose runner + /api/tasks/:id/logs endpoint"
```

---

### Task 2: acactl module + lifecycle manager

**Files:**
- Create: `cmd/acactl/go.mod`
- Create: `cmd/acactl/main.go`
- Create: `cmd/acactl/lifecycle/manager.go`
- Create: `cmd/acactl/lifecycle/ports.go`
- Modify: `go.work`

**Interfaces:**
- Produces: `lifecycle.ProcessManager` with `Start()`, `Stop()`, `IsRunning()` methods
- Produces: `lifecycle.CheckPort(port int) bool` — returns true if port is free

- [ ] **Step 1: Create go.mod**

```go
// cmd/acactl/go.mod
module adversarychef/acactl

go 1.26.4
```

Run: `go mod tidy` from cmd/acactl/

- [ ] **Step 2: Add to go.work**

```
use ./cmd/acactl
```

- [ ] **Step 3: Create lifecycle/ports.go**

```go
// cmd/acactl/lifecycle/ports.go
package lifecycle

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

func CheckPort(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("port %d is in use", port)
	}
	ln.Close()
	return nil
}

func WritePID(pidDir, name string, pid int) error {
	if pidDir == "" {
		home, _ := os.UserHomeDir()
		pidDir = home + "/.aca/pids"
	}
	os.MkdirAll(pidDir, 0755)
	return os.WriteFile(pidDir+"/"+name+".pid", []byte(strconv.Itoa(pid)), 0644)
}

func ReadPID(pidDir, name string) (int, error) {
	if pidDir == "" {
		home, _ := os.UserHomeDir()
		pidDir = home + "/.aca/pids"
	}
	data, err := os.ReadFile(pidDir + "/" + name + ".pid")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

func RemovePID(pidDir, name string) error {
	if pidDir == "" {
		home, _ := os.UserHomeDir()
		pidDir = home + "/.aca/pids"
	}
	return os.Remove(pidDir + "/" + name + ".pid")
}
```

- [ ] **Step 4: Create lifecycle/manager.go**

```go
// cmd/acactl/lifecycle/manager.go
package lifecycle

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type Service struct {
	Name    string
	Port    int
	Binary  string
	Args    []string
	Podman  bool // if true, use podman instead of binary
	cmd     *exec.Cmd
}

type ProcessManager struct {
	BinDir  string
	DataDir string
	LogDir  string
	PidDir  string
	Services []*Service
}

func (pm *ProcessManager) BuildBinary(modulePath string) error {
	cmd := exec.Command("go", "build",
		"-o", pm.BinDir+"/"+filepath.Base(modulePath),
		modulePath)
	cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (pm *ProcessManager) Start(svc *Service) error {
	if svc.Podman {
		return pm.startPodman(svc)
	}
	svc.cmd = exec.Command(svc.Binary, svc.Args...)
	logFile, _ := os.Create(pm.LogDir + "/" + svc.Name + ".log")
	svc.cmd.Stdout = logFile
	svc.cmd.Stderr = logFile
	if err := svc.cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", svc.Name, err)
	}
	WritePID(pm.PidDir, svc.Name, svc.cmd.Process.Pid)
	return nil
}

func (pm *ProcessManager) startPodman(svc *Service) error {
	// podman run -d --name <Name> --cap-add NET_RAW ... -p <Port>:<Port> <image>
	args := []string{"run", "-d", "--name", svc.Name,
		"--cap-add", "NET_RAW", "--cap-add", "NET_ADMIN",
		"--sysctl", "net.ipv6.conf.all.disable_ipv6=0",
		"-p", fmt.Sprintf("%d:%d", svc.Port, svc.Port),
		svc.Binary,
	}
	cmd := exec.Command("podman", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (pm *ProcessManager) Stop(svc *Service) error {
	if svc.Podman {
		exec.Command("podman", "stop", svc.Name).Run()
		exec.Command("podman", "rm", svc.Name).Run()
		return nil
	}
	pid, _ := ReadPID(pm.PidDir, svc.Name)
	if pid > 0 {
		p, _ := os.FindProcess(pid)
		p.Signal(os.Interrupt)
		time.Sleep(3 * time.Second)
		p.Kill()
	}
	RemovePID(pm.PidDir, svc.Name)
	return nil
}

func (pm *ProcessManager) HealthCheck(svc *Service) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/health", svc.Port)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	for ctx.Err() == nil {
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%s health check timeout", svc.Name)
}
```

Note: add `"path/filepath"` import.

- [ ] **Step 5: Build and verify**

```bash
go build ./cmd/acactl/...
```
Expected: clean build.

- [ ] **Step 6: Commit**

```bash
git add cmd/acactl/ go.work go.work.sum
git commit -m "feat(acactl): module scaffold + lifecycle manager"
```

---

### Task 3: acactl commands (up, down, status)

**Files:**
- Create: `cmd/acactl/commands/up.go`
- Create: `cmd/acactl/commands/down.go`
- Create: `cmd/acactl/commands/status.go`
- Modify: `cmd/acactl/main.go`

**Interfaces:**
- Consumes: `lifecycle.ProcessManager`, `lifecycle.Service`, `lifecycle.CheckPort`
- Produces: CLI flags `aca up`, `aca down`, `aca status`

- [ ] **Step 1: Create commands/up.go**

```go
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
	if dataDir == "" { dataDir = home + "/.aca" }

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
	pm.BuildBinary("./servers/nexus/cmd/server", filepath.Join(pm.BinDir, "nexus-mcp"))
	fmt.Println("Building acasched...")
	pm.BuildBinary("./cmd/acasched", filepath.Join(pm.BinDir, "acasched"))

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
	pm.Start(nexus)
	pm.HealthCheck(nexus)

	// Start kali-mcp
	fmt.Println("Starting kali-mcp...")
	kali := &lifecycle.Service{
		Name:   "kali-mcp",
		Port:   kaliPort,
		Binary: "kali-mcp",
		Podman: true,
	}
	pm.Start(kali)
	pm.HealthCheck(kali)

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
	pm.Start(acasched)
	pm.HealthCheck(acasched)

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
```

Note: update `BuildBinary` to accept output path parameter, or adjust the call. Add proper error handling for build failures.

- [ ] **Step 2: Create commands/down.go**

```go
// cmd/acactl/commands/down.go
package commands

import (
	"fmt"
	"os"

	"adversarychef/acactl/lifecycle"
)

func Down(dataDir string, ports [3]int) error {
	home, _ := os.UserHomeDir()
	if dataDir == "" { dataDir = home + "/.aca" }

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
```

- [ ] **Step 3: Create commands/status.go**

```go
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
```

- [ ] **Step 4: Create main.go with cobra-style flag parsing**

```go
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
```

- [ ] **Step 5: Build**

```bash
go build -o /tmp/acactl ./cmd/acactl/
```
Expected: clean build, binary at /tmp/acactl.

- [ ] **Step 6: Commit**

```bash
git add cmd/acactl/
git commit -m "feat(acactl): up/down/status commands"
```

---

### Task 4: acactl display + logs + tasks commands

**Files:**
- Create: `cmd/acactl/display/table.go`
- Create: `cmd/acactl/display/stream.go`
- Create: `cmd/acactl/commands/tasks.go`
- Create: `cmd/acactl/commands/logs.go`
- Create: `cmd/acactl/commands/projects.go`
- Modify: `cmd/acactl/main.go`

**Interfaces:**
- Consumes: `display.FormatStreamJSON(raw string)` — formats stream-json lines
- Consumes: `GET /api/tasks?project_id=X&status=Y`, `GET /api/tasks/{id}/logs`

- [ ] **Step 1: Create display/table.go**

```go
// cmd/acactl/display/table.go
package display

import "fmt"

func PrintTaskTable(tasks []TaskSummary) {
	fmt.Println("┌──────────────────────┬──────────┬─────────┬──────────────────────┬──────────┐")
	fmt.Println("│ TASK ID              │ AGENT    │ STATUS  │ TITLE                │ DURATION │")
	fmt.Println("├──────────────────────┼──────────┼─────────┼──────────────────────┼──────────┤")
	for _, t := range tasks {
		fmt.Printf("│ %-20s │ %-8s │ %-7s │ %-20s │ %-8s │\n",
			truncate(t.ID, 20), truncate(t.Agent, 8),
			t.Status, truncate(t.Title, 20), t.Duration)
	}
	fmt.Println("└──────────────────────┴──────────┴─────────┴──────────────────────┴──────────┘")
}

func truncate(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n-3] + "..."
}

type TaskSummary struct {
	ID, Agent, Status, Title, Duration string
}
```

- [ ] **Step 2: Create display/stream.go**

```go
// cmd/acactl/display/stream.go
package display

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type StreamLine struct {
	Type   string `json:"type"`
	Name   string `json:"name,omitempty"`
	Text   string `json:"text,omitempty"`
	Result string `json:"result,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

func FormatStreamJSON(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var line StreamLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		switch line.Type {
		case "tool_call":
			fmt.Printf("  ▸ [mcp] %s\n", line.Name)
			if line.Params != nil {
				printParams(line.Params)
			}
		case "tool_result":
			printResult(line.Result)
		case "assistant":
			text := strings.TrimSpace(line.Text)
			if text != "" {
				fmt.Printf("\n  ✦ %s\n\n", wordWrap(text, 72))
			}
		}
	}
	return nil
}

func printParams(raw json.RawMessage) {
	var m map[string]interface{}
	json.Unmarshal(raw, &m)
	for k, v := range m {
		fmt.Printf("    %s: %v\n", k, v)
	}
}

func printResult(raw string) {
	lines := strings.Split(raw, "\n")
	n := len(lines)
	if n > 200 {
		lines = lines[:200]
	}
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			fmt.Printf("    %s\n", l)
		}
	}
	if n > 200 {
		fmt.Printf("    [truncated, -%d lines]\n", n-200)
	}
}

func wordWrap(s string, width int) string {
	if len(s) <= width { return s }
	return s[:width-3] + "..."
}
```

- [ ] **Step 3: Create commands/tasks.go**

```go
// cmd/acactl/commands/tasks.go
package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"adversarychef/acactl/display"
)

func Tasks(acaPort int, projectID, status string) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/tasks", acaPort)
	if projectID != "" { url += "?project_id=" + projectID }
	if status != "" { url += "&status=" + status }

	resp, err := http.Get(url)
	if err != nil { return fmt.Errorf("acasched unreachable: %w", err) }
	defer resp.Body.Close()

	var tasks []struct {
		ID, Agent, Status, Title string
	}
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return fmt.Errorf("parse tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	var summaries []display.TaskSummary
	for _, t := range tasks {
		summaries = append(summaries, display.TaskSummary{
			ID: t.ID, Agent: t.Agent, Status: t.Status, Title: t.Title,
		})
	}
	display.PrintTaskTable(summaries)
	return nil
}
```

- [ ] **Step 4: Create commands/logs.go**

```go
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
	if follow { url += "?follow=true" }

	resp, err := http.Get(url)
	if err != nil { return fmt.Errorf("acasched unreachable: %w", err) }
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
```

- [ ] **Step 5: Create commands/projects.go**

```go
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
	if err != nil { return fmt.Errorf("acasched unreachable: %w", err) }
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("Project created: %s\n", result["id"])
	return nil
}
```

- [ ] **Step 6: Update main.go to dispatch new commands**

Add to the switch in main.go:

```go
case "tasks":
	project := flag.String("project", "", "filter by project ID")
	st := flag.String("status", "", "filter by status")
	flag.CommandLine.Parse(os.Args[2:])
	commands.Tasks(ports[2], *project, *st)
case "logs":
	taskID := flag.String("task", "", "task ID")
	follow := flag.Bool("follow", false, "follow streaming output")
	raw := flag.Bool("raw", false, "output raw stream-json")
	flag.CommandLine.Parse(os.Args[2:])
	commands.Logs(ports[2], *taskID, *follow, *raw)
```

- [ ] **Step 7: Build and verify**

```bash
go build -o /tmp/acactl ./cmd/acactl/
/tmp/acactl --help
```
Expected: help output showing all commands.

- [ ] **Step 8: Commit**

```bash
git add cmd/acactl/
git commit -m "feat(acactl): display engine + logs/tasks/projects commands"
```
