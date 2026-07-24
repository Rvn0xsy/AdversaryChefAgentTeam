// Package job provides async command execution and job management.
package job

import (
	"fmt"
	"sync"
	"time"
)

// Status represents a job status.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusKilled    Status = "killed"
	StatusTimedOut  Status = "timed_out"
)

// Job represents an asynchronously executed task.
type Job struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	Status    Status    `json:"status"`
	Stdout    string    `json:"stdout"`
	Stderr    string    `json:"stderr"`
	ExitCode  int       `json:"exit_code"`
	CreatedAt time.Time `json:"created_at"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`

	mu           sync.RWMutex
	kill         func()        // force-kills the process group
	done         chan struct{} // closed when the job finishes
	killedByUser bool          // set when Kill is called externally
}

// Snapshot returns a thread-safe copy of the job.
func (j *Job) Snapshot() Job {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return Job{
		ID:        j.ID,
		Command:   j.Command,
		Status:    j.Status,
		Stdout:    j.Stdout,
		Stderr:    j.Stderr,
		ExitCode:  j.ExitCode,
		CreatedAt: j.CreatedAt,
		StartedAt: j.StartedAt,
		EndedAt:   j.EndedAt,
	}
}

// appendOutput appends output in a thread-safe manner.
func (j *Job) appendOutput(stdout, stderr string, maxOutput int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Stdout = truncateAppend(j.Stdout, stdout, maxOutput)
	j.Stderr = truncateAppend(j.Stderr, stderr, maxOutput)
}

// setRunning marks the job as running.
func (j *Job) setRunning() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = StatusRunning
	j.StartedAt = time.Now()
}

// setFinished marks the job as finished.
func (j *Job) setFinished(exitCode int, timedOut, killed bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.ExitCode = exitCode
	j.EndedAt = time.Now()
	switch {
	case killed:
		j.Status = StatusKilled
	case timedOut:
		j.Status = StatusTimedOut
	case exitCode == 0:
		j.Status = StatusCompleted
	default:
		j.Status = StatusFailed
	}
}

func (j *Job) markKilledByUser() {
	j.mu.Lock()
	j.killedByUser = true
	j.mu.Unlock()
}

func (j *Job) isKilledByUser() bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.killedByUser
}

func truncateAppend(existing, added string, max int) string {
	combined := existing + added
	if len(combined) > max {
		return combined[:max] + fmt.Sprintf("\n... [output truncated at %d bytes]", max)
	}
	return combined
}
