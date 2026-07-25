// cmd/acactl/display/stream.go
package display

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type StreamLine struct {
	Type   string          `json:"type"`
	Name   string          `json:"name,omitempty"`
	Text   string          `json:"text,omitempty"`
	Result string          `json:"result,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

func FormatStreamJSON(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var line StreamLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		switch line.Type {
		case "tool_call":
			fmt.Printf("  ▸ [mcp] %s\n", line.Name)
			if line.Params != nil {
				printParams(line.Params)
			}
		case "tool_result":
			printResult(line.Result)
		case "assistant":
			text := strings.TrimSpace(line.Text)
			if text != "" {
				fmt.Printf("\n  ✦ %s\n\n", wordWrap(text, 72))
			}
		}
	}
	return nil
}

func printParams(raw json.RawMessage) {
	var m map[string]interface{}
	json.Unmarshal(raw, &m)
	for k, v := range m {
		fmt.Printf("    %s: %v\n", k, v)
	}
}

func printResult(raw string) {
	lines := strings.Split(raw, "\n")
	n := len(lines)
	if n > 200 {
		lines = lines[:200]
	}
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			fmt.Printf("    %s\n", l)
		}
	}
	if n > 200 {
		fmt.Printf("    [truncated, -%d lines]\n", n-200)
	}
}

func wordWrap(s string, width int) string {
	if len(s) <= width {
		return s
	}
	return s[:width-3] + "..."
}
