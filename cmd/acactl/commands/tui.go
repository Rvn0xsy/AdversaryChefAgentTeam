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
