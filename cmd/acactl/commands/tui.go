package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"adversarychef/acactl/display"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Messages ──

type taskAppearedMsg struct{ task taskEntry }
type taskStatusMsg   struct{ id, status string }
type logLineMsg      struct{ taskID, line string }
type pollTickMsg     struct{}

// ── Model ──

type taskEntry struct {
	ID, Agent, Title, Status string
}

type logBuf struct {
	lines   []string
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

type tuiModel struct {
	client      *http.Client
	baseURL     string
	projectID   string
	projectName string

	tasks    []taskEntry
	selected int
	logs     map[string]*logBuf
	err      error

	width, height int
	quitting      bool
}

func (m *tuiModel) Init() tea.Cmd {
	return pollTick()
}

func pollTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

// channelMsg wraps a channel for continuous log streaming.
type channelMsg struct {
	ch     chan tea.Msg
	taskID string
}

// streamLogCmd returns a tea.Cmd that opens an SSE connection for taskID
// and continuously streams log lines via channelMsg. Each formatted line
// is sent as a logLineMsg, and the channel is re-dispatched so Update
// keeps draining.
func streamLogCmd(client *http.Client, baseURL, taskID string) tea.Cmd {
	return func() tea.Msg {
		url := baseURL + "/api/tasks/" + taskID + "/logs?follow=true"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return logLineMsg{taskID: taskID, line: fmt.Sprintf("[error: %v]", err)}
		}
		resp, err := client.Do(req)
		if err != nil {
			return logLineMsg{taskID: taskID, line: fmt.Sprintf("[error: %v]", err)}
		}

		ch := make(chan tea.Msg, 100)
		go func() {
			defer resp.Body.Close()
			defer close(ch)
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.HasPrefix(line, "data: ") {
					continue
				}
				payload := strings.TrimPrefix(line, "data: ")

				var header struct{ Type string `json:"type"` }
				if json.Unmarshal([]byte(payload), &header) == nil && header.Type == "complete" {
					ch <- logLineMsg{taskID: taskID, line: "  ✅ Completed"}
					return
				}

				lines := display.FormatMessage([]byte(payload))
				for _, l := range lines {
					ch <- logLineMsg{taskID: taskID, line: l}
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				ch <- logLineMsg{taskID: taskID, line: fmt.Sprintf("[stream error: %v]", err)}
			}
		}()
		return channelMsg{ch: ch, taskID: taskID}
	}
}

// pollTasksCmd fetches the project task list and returns taskAppearedMsg for each.
func pollTasksCmd(client *http.Client, baseURL, projectID string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(200 * time.Millisecond) // brief backoff
		resp, err := client.Get(baseURL + "/api/projects/" + projectID + "/tasks")
		if err != nil {
			return pollTickMsg{} // retry next tick
		}
		defer resp.Body.Close()

		var tasks []struct {
			ID     string `json:"id"`
			Agent  string `json:"agent"`
			Status string `json:"status"`
			Title  string `json:"title"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
			return pollTickMsg{}
		}

		var msgs []tea.Cmd
		for _, t := range tasks {
			t := t // capture loop variable
			msgs = append(msgs, func() tea.Msg {
				return taskAppearedMsg{
					task: taskEntry{
						ID:     t.ID,
						Agent:  strings.TrimPrefix(t.Agent, "red-team/"),
						Title:  t.Title,
						Status: t.Status,
					},
				}
			})
		}
		if len(msgs) == 0 {
			return pollTickMsg{}
		}
		return tea.Batch(msgs...)
	}
}

// ── Update ──

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			for _, lb := range m.logs {
				if lb.cancel != nil {
					lb.cancel()
				}
			}
			return m, tea.Quit

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}

		case "down", "j":
			if m.selected < len(m.tasks)-1 {
				m.selected++
			}

		case "tab":
			for i := 1; i <= len(m.tasks); i++ {
				idx := (m.selected + i) % len(m.tasks)
				if m.tasks[idx].Status == "running" || m.tasks[idx].Status == "pending" || m.tasks[idx].Status == "dispatched" {
					m.selected = idx
					break
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case pollTickMsg:
		cmds = append(cmds, pollTasksCmd(m.client, m.baseURL, m.projectID))
		cmds = append(cmds, pollTick())
		if m.allTasksDone() {
			cmds = append(cmds, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return tea.Quit()
			}))
		}

	case taskAppearedMsg:
		found := false
		for i, t := range m.tasks {
			if t.ID == msg.task.ID {
				m.tasks[i].Status = msg.task.Status
				found = true
				break
			}
		}
		if !found {
			m.tasks = append(m.tasks, msg.task)
			ctx, cancel := context.WithCancel(context.Background())
			m.logs[msg.task.ID] = &logBuf{
				running: true,
				cancel:  cancel,
			}
			_ = ctx
			cmds = append(cmds, streamLogCmd(m.client, m.baseURL, msg.task.ID))
		}
		if msg.task.Status == "done" || msg.task.Status == "failed" || msg.task.Status == "timeout" {
			if lb, ok := m.logs[msg.task.ID]; ok {
				lb.running = false
			}
		}

	case logLineMsg:
		if lb, ok := m.logs[msg.taskID]; ok {
			lb.mu.Lock()
			lb.lines = append(lb.lines, msg.line)
			if len(lb.lines) > 500 {
				lb.lines = lb.lines[len(lb.lines)-500:]
			}
			lb.mu.Unlock()
		}

	case channelMsg:
		drained := false
		for !drained {
			select {
			case inner, ok := <-msg.ch:
				if !ok {
					drained = true
				} else {
					cmds = append(cmds, func() tea.Msg { return inner })
				}
			default:
				drained = true
			}
		}
		cmds = append(cmds, func() tea.Msg { return msg })
	}

	return m, tea.Batch(cmds...)
}

func (m *tuiModel) allTasksDone() bool {
	if len(m.tasks) == 0 {
		return false
	}
	for _, t := range m.tasks {
		switch t.Status {
		case "running", "pending", "dispatched":
			return false
		}
	}
	return true
}

// ── View ──

var statusIcons = map[string]string{
	"running":    "🟡",
	"pending":    "⏳",
	"dispatched": "📤",
	"done":       "✅",
	"failed":     "❌",
	"timeout":    "⏰",
}

func (m *tuiModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	w := m.width
	if w < 40 {
		w = 80 // fallback before first WindowSizeMsg
	}

	// ── Top bar ──
	title := fmt.Sprintf("Project: %s", m.projectName)
	padding := w - len(title) - 40
	if padding < 2 {
		padding = 2
	}
	b.WriteString(fmt.Sprintf("┌─ %s %s[q]退出 [↑↓]选择 [Tab]下一个 ─┐\n",
		title, strings.Repeat(" ", padding)))
	b.WriteByte('\n')

	// ── Task list ──
	for i, t := range m.tasks {
		marker := " "
		if i == m.selected {
			marker = "▶"
		}
		icon := statusIcons[t.Status]
		if icon == "" {
			icon = "  "
		}
		line := fmt.Sprintf("  %s %-16s %s %-9s │  %s",
			marker, t.Agent, icon, t.Status, t.Title)
		if len(line) > w-2 {
			line = line[:w-2]
		}
		b.WriteString(line)
		b.WriteString(strings.Repeat(" ", max(0, w-2-len(line))))
		b.WriteString("  \n")
	}
	b.WriteByte('\n')

	// ── Separator ──
	b.WriteString(strings.Repeat("─", w))
	b.WriteByte('\n')

	// ── Log area ──
	if m.selected < len(m.tasks) {
		sel := m.tasks[m.selected]
		icon := statusIcons[sel.Status]
		if icon == "" {
			icon = "  "
		}
		header := fmt.Sprintf("  [%s]  %s %s", sel.Agent, icon, sel.Status)
		b.WriteString(header)
		b.WriteByte('\n')

		if lb, ok := m.logs[sel.ID]; ok {
			lb.mu.Lock()
			lines := make([]string, len(lb.lines))
			copy(lines, lb.lines)
			lb.mu.Unlock()

			logHeight := m.height - 7 - len(m.tasks)
			if logHeight < 5 {
				logHeight = 5
			}
			start := 0
			if len(lines) > logHeight {
				start = len(lines) - logHeight
			}
			for _, l := range lines[start:] {
				trimmed := l
				if len(trimmed) > w-2 {
					trimmed = trimmed[:w-2]
				}
				b.WriteString("  " + trimmed)
				b.WriteByte('\n')
			}
		}
	}

	b.WriteString(strings.Repeat("─", w))
	b.WriteByte('\n')
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
