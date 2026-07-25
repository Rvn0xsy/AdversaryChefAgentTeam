package api

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
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
		// Wait for file to appear (task just started, log not written yet)
		for i := 0; i < 30; i++ {
			time.Sleep(500 * time.Millisecond)
			f, err = os.Open(logPath)
			if err == nil {
				break
			}
			// Client disconnected?
			select {
			case <-r.Context().Done():
				return
			default:
			}
		}
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			return
		}
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	for {
		line, err := reader.ReadBytes('\n')
		if err == nil {
			fmt.Fprintf(w, "data: %s\n\n", line[:len(line)-1]) // strip newline
			flusher.Flush()
			continue
		}

		if err == io.EOF {
			// Wait for more data or client disconnect
			select {
			case <-r.Context().Done():
				return
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}

		// Real error
		return
	}
}