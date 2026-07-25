package goose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"adversarychef/acasched/internal/store"
)

type Runner struct {
	PromptsDir string
	WorkDir    string
	LogDir     string
	NexusMCP   string
	KaliMCP    string
	MythicMCP  string
}

func (r *Runner) Execute(ctx context.Context, task *store.Task) (*Result, error) {
	prompt := r.buildPrompt(task)
	tmpFile, err := os.CreateTemp("", "goose-instructions-*.md")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	if _, err := tmpFile.WriteString(prompt); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	args := []string{
		"run",
		"--instructions", tmpFile.Name(),
		"--text", task.Description,
		"--max-turns", fmt.Sprintf("%d", task.MaxTurns),
		"--no-session",
		"--output-format", "stream-json",
		"--no-profile",
	}
	if r.NexusMCP != "" {
		args = append(args, "--with-streamable-http-extension", r.NexusMCP)
	}
	if r.KaliMCP != "" {
		args = append(args, "--with-streamable-http-extension", r.KaliMCP)
	}
	if r.MythicMCP != "" {
		args = append(args, "--with-streamable-http-extension", r.MythicMCP)
	}

	cmd := exec.CommandContext(ctx, "goose", args...)
	if r.WorkDir != "" {
		cmd.Dir = r.WorkDir
	}

	if r.LogDir != "" {
		os.MkdirAll(r.LogDir, 0755)
		logPath := filepath.Join(r.LogDir, task.ID+".jsonl")
		f, err := os.Create(logPath)
		if err != nil {
			return nil, fmt.Errorf("create log file: %w", err)
		}
		defer f.Close()

		// Stderr goes directly to the log file
		cmd.Stderr = f

		// Pipe stdout: write to file + capture for parsing
		pr, pw, _ := os.Pipe()
		cmd.Stdout = pw

		var buf bytes.Buffer
		done := make(chan struct{})
		go func() {
			io.Copy(io.MultiWriter(f, &buf), pr)
			close(done)
		}()

		err = cmd.Run()
		pw.Close() // signal EOF to the pipe reader
		<-done     // wait for the copy to complete
		if err != nil {
			return nil, fmt.Errorf("goose exited: %w", err)
		}
		return parseStreamOutput(buf.String()), nil
	}

	// Fallback when LogDir is empty (existing behavior)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("goose exited: %w, output: %s", err, string(output))
	}
	return parseStreamOutput(string(output)), nil
}

func (r *Runner) buildPrompt(task *store.Task) string {
	content, err := os.ReadFile(r.PromptsDir + "/" + task.Agent + ".md")
	agentPrompt := ""
	if err == nil {
		agentPrompt = string(content)
	}

	return fmt.Sprintf(`## Session Binding
project_id: %s
task_id: %s

## Task Lifecycle
- Use scheduler_create_task to delegate work
- Use scheduler_complete_task to mark yourself done
- Do NOT exit without calling scheduler_complete_task

---
%s`, task.ProjectID, task.ID, agentPrompt)
}

type Result struct {
	Status  string
	Summary string
	Output  string
}
