package api

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
		flusher.Flush()
	}
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}
