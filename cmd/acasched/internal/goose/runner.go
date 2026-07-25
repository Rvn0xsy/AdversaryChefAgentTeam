package goose

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"adversarychef/acasched/internal/store"
)

type Runner struct {
	PromptsDir string                 // e.g., "prompts"
	SkillsDir  string                 // e.g., "skills"
	LogDir     string
	Registry   map[string]string      // loaded from _mcp-registry.yaml
}

func (r *Runner) Execute(ctx context.Context, task *store.Task) (*Result, error) {
	// Parse agent path: task.Agent format "red-team/echo-recon"
	parts := strings.SplitN(task.Agent, "/", 2)
	agentPath := task.Agent + ".md"
	if len(parts) == 2 {
		agentPath = filepath.Join(r.PromptsDir, parts[0], parts[1]+".md")
	}
	agentPromptBytes, err := os.ReadFile(agentPath)
	var agentPromptStr string
	if err != nil {
		log.Printf("runner: failed to read prompt %s: %v", agentPath, err)
	} else {
		agentPromptStr = string(agentPromptBytes)
	}

	prompt := r.buildPrompt(task, agentPromptStr)
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
		"--max-turns", fmt.Sprintf("%d", task.MaxTurns),
		"--no-session",
		"--output-format", "stream-json",
		"--no-profile",
	}

	// Parse prompt for MCP requirements
	meta := ParsePromptMeta(agentPromptBytes)

	// Mount only required MCP extensions from registry
	for _, mcpName := range meta.Requires {
		url, ok := r.Registry[mcpName]
		if !ok {
			log.Printf("runner: MCP %q not found in registry, skipping", mcpName)
			continue
		}
		args = append(args, "--with-streamable-http-extension", url)
	}

	// Mount _shared skills (always)
	sharedSkillsDir := filepath.Join(r.SkillsDir, "_shared")
	if entries, err := os.ReadDir(sharedSkillsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			hostPath := filepath.Join(sharedSkillsDir, entry.Name())
			containerPath := "/root/.agents/skills/" + entry.Name()
			args = append(args, "-v", hostPath+":"+containerPath+":ro")
		}
	}

	// Mount agent-specific skills from prompt metadata
	for _, skill := range meta.Skills {
		hostPath := filepath.Join(r.SkillsDir, skill)
		if _, err := os.Stat(hostPath); os.IsNotExist(err) {
			log.Printf("runner: skill dir %q not found, skipping", hostPath)
			continue
		}
		skillName := filepath.Base(skill)
		containerPath := "/root/.agents/skills/" + skillName
		args = append(args, "-v", hostPath+":"+containerPath+":ro")
	}

	cmd := exec.CommandContext(ctx, "goose", args...)

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

func (r *Runner) buildPrompt(task *store.Task, agentPrompt string) string {
	return fmt.Sprintf(`## Session Binding
project_id: %s
task_id: %s

## Task
%s

## Task Lifecycle
- Use scheduler_create_task to delegate work
- Use scheduler_complete_task to mark yourself done
- Do NOT exit without calling scheduler_complete_task

---
%s`, task.ProjectID, task.ID, task.Description, agentPrompt)
}

type Result struct {
	Status  string
	Summary string
	Output  string
}
