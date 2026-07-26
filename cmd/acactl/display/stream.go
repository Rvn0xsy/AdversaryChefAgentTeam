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
	Type    string          `json:"type"`
	Message *struct {
		Content json.RawMessage `json:"content"`
	} `json:"message,omitempty"`
}

func FormatStreamJSON(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var line StreamLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		switch line.Type {
		case "message":
			lines := FormatMessage(scanner.Bytes())
			for _, l := range lines {
				fmt.Println(l)
			}
		case "complete":
			fmt.Println("  ✅ Completed")
		}
	}
	return nil
}

// FormatMessage parses a single stream-json line and returns formatted log lines.
// Used by both TUI (via logLineMsg) and the logs command for consistent output.
func FormatMessage(rawJSON []byte) []string {
	var line StreamLine
	if err := json.Unmarshal(rawJSON, &line); err != nil {
		return nil
	}
	if line.Type != "message" || line.Message == nil {
		return nil
	}

	var blocks []struct {
		Type         string `json:"type"`
		Text         string `json:"text"`
		Thinking     string `json:"thinking"`
		ToolCall     *struct {
			Status string `json:"status"`
			Value  struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			} `json:"value"`
		} `json:"toolCall,omitempty"`
		ToolResponse *struct {
			Status string `json:"status"`
			Value  string `json:"value"`
		} `json:"toolResponse,omitempty"`
		Meta *struct {
			GooseExtension string `json:"goose_extension"`
		} `json:"_meta,omitempty"`
	}
	if err := json.Unmarshal(line.Message.Content, &blocks); err != nil {
		return nil
	}

	var result []string
	var textBuf strings.Builder

	for _, b := range blocks {
		switch b.Type {
		case "thinking":
		case "text":
			textBuf.WriteString(b.Text)
		case "toolRequest":
			if textBuf.Len() > 0 {
				t := strings.TrimSpace(textBuf.String())
				if t != "" {
					result = append(result, "  ✦ "+t)
				}
				textBuf.Reset()
			}
			if b.ToolCall != nil {
				server := ""
				if b.Meta != nil {
					server = b.Meta.GooseExtension
				}
				name := b.ToolCall.Value.Name
				if idx := strings.Index(name, "__"); idx > 0 {
					if server == "" {
						server = name[:idx]
					}
					name = name[idx+2:]
				}
				result = append(result, fmt.Sprintf("  ▸ [%s] %s", server, name))
				for k, v := range b.ToolCall.Value.Arguments {
					result = append(result, fmt.Sprintf("    %s: %v", k, v))
				}
			}
		case "toolResponse":
			if b.ToolResponse != nil && b.ToolResponse.Value != "" {
				val := b.ToolResponse.Value
				if len(val) > 500 {
					val = val[:500] + "..."
				}
				result = append(result, "    → "+val)
			}
		}
	}

	if textBuf.Len() > 0 {
		t := strings.TrimSpace(textBuf.String())
		if t != "" {
			result = append(result, "  ✦ "+t)
		}
	}

	return result
}

