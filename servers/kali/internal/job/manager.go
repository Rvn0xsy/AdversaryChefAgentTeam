package job

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Manager manages the lifecycle of all async jobs.
type Manager struct {
	mu             sync.RWMutex
	jobs           map[string]*Job
	maxOutput      int
	defaultTimeout time.Duration
}

// JobSummary is a lightweight job view without stdout，without stdout/stderr.
type JobSummary struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	Status    Status    `json:"status"`
	ExitCode  int       `json:"exit_code"`
	CreatedAt time.Time `json:"created_at"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}

const (
	DefaultMaxOutput = 500_000
	DefaultTimeout   = 30 * time.Minute
)

// NewManager creates a new job manager.
func NewManager(maxOutput int, defaultTimeout time.Duration) *Manager {
	return &Manager{
		jobs:           make(map[string]*Job),
		maxOutput:      maxOutput,
		defaultTimeout: defaultTimeout,
	}
}

// Start creates and starts an async job, returning its ID.
func (m *Manager) Start(command string, timeout time.Duration) string {
	if timeout <= 0 {
		timeout = m.defaultTimeout
	}

	j := &Job{
		ID:        genJobID(),
		Command:   command,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		done:      make(chan struct{}),
	}

	m.mu.Lock()
	m.jobs[j.ID] = j
	m.mu.Unlock()

	go m.run(j, timeout)
	return j.ID
}

// Get returns a job snapshot including live output.including live output.
func (m *Manager) Get(id string) (*Job, error) {
	m.mu.RLock()
	j, ok := m.jobs[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	snapshot := j.Snapshot()
	return &snapshot, nil
}

// List returns job summaries(without stdout/stderr), optionally filtered by status.
func (m *Manager) List(statusFilter Status) []JobSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []JobSummary
	for _, j := range m.jobs {
		s := j.Snapshot()
		if statusFilter == "" || s.Status == statusFilter {
			out = append(out, JobSummary{
				ID:        s.ID,
				Command:   s.Command,
				Status:    s.Status,
				ExitCode:  s.ExitCode,
				CreatedAt: s.CreatedAt,
				StartedAt: s.StartedAt,
				EndedAt:   s.EndedAt,
			})
		}
	}
	return out
}

// Kill terminates a job: mark killedByUser → force-kill process group → wait for run goroutine.
func (m *Manager) Kill(id string) error {
	m.mu.RLock()
	j, ok := m.jobs[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	j.mu.RLock()
	status := j.Status
	killFn := j.kill
	j.mu.RUnlock()

	if status != StatusPending && status != StatusRunning {
		return fmt.Errorf("job %s is already %s", id, status)
	}

	j.markKilledByUser()

	if killFn != nil {
		killFn()
	}

	<-j.done
	return nil
}

// run executes the command in a goroutine with streaming output via pipes.Output is streamed via StdoutPipe/StderrPipe.

func (m *Manager) run(j *Job, timeout time.Duration) {
	defer close(j.done)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.Command("bash", "-c", j.Command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		j.appendOutput("", fmt.Sprintf("failed to create stdout pipe: %v", err), m.maxOutput)
		j.setFinished(-1, false, false)
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		j.appendOutput("", fmt.Sprintf("failed to create stderr pipe: %v", err), m.maxOutput)
		j.setFinished(-1, false, false)
		return
	}

	killProc := func() {
		if cmd.Process != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}

	j.mu.Lock()
	j.kill = killProc
	j.mu.Unlock()

	j.setRunning()

	if err := cmd.Start(); err != nil {
		j.appendOutput("", fmt.Sprintf("failed to start: %v", err), m.maxOutput)
		j.setFinished(-1, false, false)
		return
	}

	// start two goroutines to stream stdout stdout/stderr into the job
	var readers sync.WaitGroup
	readers.Add(2)
	go streamPipe(stdoutPipe, func(chunk string) {
		j.appendOutput(chunk, "", m.maxOutput)
	}, &readers)
	go streamPipe(stderrPipe, func(chunk string) {
		j.appendOutput("", chunk, m.maxOutput)
	}, &readers)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	var (
		timedOut bool
		exitCode int
	)

	select {
	case err := <-waitDone:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}

	case <-ctx.Done():
		killProc()
		<-waitDone
		timedOut = true
	}

	// wait for reader goroutines to drain remaining pipe data
	readers.Wait()

	killed := j.isKilledByUser()

	if killed {
		j.setFinished(-1, false, true)
	} else {
		j.setFinished(exitCode, timedOut, false)
	}
}

// streamPipe reads line-by-line from a reader, calling writeFn for each line.
func streamPipe(r io.Reader, writeFn func(string), wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line buffer
	for scanner.Scan() {
		writeFn(scanner.Text() + "\n")
	}
	// scanner exits on EOF or pipe close, ignoring errors
}

func genJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}
