package job

import (
	"strings"
	"testing"
	"time"
)

func TestStartAndGet(t *testing.T) {
	mgr := NewManager(10000, 10*time.Second)
	id := mgr.Start("echo hello", 0)

	j, err := mgr.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if j.Command != "echo hello" {
		t.Fatalf("unexpected command: %s", j.Command)
	}

	time.Sleep(500 * time.Millisecond)

	j, err = mgr.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if j.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s, stdout=%s stderr=%s", j.Status, j.Stdout, j.Stderr)
	}
	if !strings.Contains(j.Stdout, "hello") {
		t.Fatalf("expected 'hello' in stdout, got: %s", j.Stdout)
	}
}

func TestStartKill(t *testing.T) {
	mgr := NewManager(10000, 10*time.Second)
	id := mgr.Start("sleep 30", 0)

	time.Sleep(200 * time.Millisecond)

	j, _ := mgr.Get(id)
	if j.Status != StatusRunning && j.Status != StatusPending {
		t.Fatalf("expected running/pending, got %s", j.Status)
	}

	if err := mgr.Kill(id); err != nil {
		t.Fatal(err)
	}

	j, _ = mgr.Get(id)
	if j.Status != StatusKilled {
		t.Fatalf("expected killed, got %s", j.Status)
	}
}

func TestStartTimeout(t *testing.T) {
	mgr := NewManager(10000, 200*time.Millisecond)
	id := mgr.Start("sleep 10", 0)

	time.Sleep(800 * time.Millisecond)

	j, _ := mgr.Get(id)
	if j.Status != StatusTimedOut {
		t.Fatalf("expected timed_out, got %s", j.Status)
	}
}

func TestList(t *testing.T) {
	mgr := NewManager(10000, 10*time.Second)
	_ = mgr.Start("sleep 1", 0)
	_ = mgr.Start("echo a", 0)

	time.Sleep(500 * time.Millisecond)

	all := mgr.List("")
	if len(all) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(all))
	}
	for _, s := range all {
		if s.ID == "" {
			t.Fatal("job id is empty")
		}
		if s.Command == "" {
			t.Fatal("job command is empty")
		}
	}

	running := mgr.List(StatusRunning)
	if len(running) < 1 {
		t.Fatalf("expected at least 1 running, got %d", len(running))
	}
}

func TestKillNonexistent(t *testing.T) {
	mgr := NewManager(10000, 10*time.Second)
	err := mgr.Kill("job_nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent job")
	}
}

func TestStreamingOutput(t *testing.T) {
	mgr := NewManager(100000, 10*time.Second)
	// Command: output one line, sleep, output another — ensures partial output is readable mid-run
	id := mgr.Start("echo 'line1'; sleep 1; echo 'line2'", 0)

	// wait briefly for the first line to appear
	time.Sleep(200 * time.Millisecond)

	j, err := mgr.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if j.Status != StatusRunning {
		t.Fatalf("expected running, got %s", j.Status)
	}
	if j.Stdout == "" {
		t.Fatal("expected partial stdout while running, got empty")
	}
	t.Logf("partial stdout: %q", j.Stdout)

	// wait for completion
	time.Sleep(1200 * time.Millisecond)

	j, _ = mgr.Get(id)
	if j.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", j.Status)
	}
	t.Logf("final stdout: %q", j.Stdout)
}
