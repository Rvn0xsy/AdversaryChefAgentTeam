package goose

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"adversarychef/acasched/internal/store"
)

type Runner struct {
	PromptsDir string
	WorkDir    string
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
