package commands

import (
	"context"
	_ "encoding/json" // for future use
	_ "fmt"           // for future use
	"net/http"
	_ "strings" // for future use
	"sync"
	"time"

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
