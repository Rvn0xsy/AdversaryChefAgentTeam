package goose

import (
	"bufio"
	"encoding/json"
	"strings"
)

type streamLine struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

func parseStreamOutput(output string) *Result {
	var summary strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		var line streamLine
		if json.Unmarshal(scanner.Bytes(), &line) != nil {
			continue
		}
		if line.Type == "assistant" {
			var blocks []struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(line.Content, &blocks) == nil {
				for _, b := range blocks {
					summary.WriteString(b.Text)
				}
			}
		}
	}
	return &Result{Status: "done", Summary: summary.String(), Output: output}
}
