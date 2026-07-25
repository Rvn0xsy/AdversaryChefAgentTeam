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
	var textBuf strings.Builder
	scanner := bufio.NewScanner(r)
	flushText := func() {
		t := strings.TrimSpace(textBuf.String())
		if t != "" {
			fmt.Printf("\n  ✦ %s\n\n", t)
			textBuf.Reset()
		}
	}
	for scanner.Scan() {
		var line StreamLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		switch line.Type {
		case "message":
			if line.Message != nil {
				formatMessageContent(line.Message.Content, &textBuf, flushText)
			}
		case "complete":
			flushText()
			fmt.Println("  ✅ Completed")
		}
	}
	flushText()
	return nil
}

func formatMessageContent(raw json.RawMessage, textBuf *strings.Builder, flushText func()) {
	var blocks []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Thinking string `json:"thinking"`
		ToolCall *struct {
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
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return
	}
	for _, b := range blocks {
		switch b.Type {
		case "thinking":
			// skip
		case "text":
			textBuf.WriteString(b.Text)
		case "toolRequest":
			flushText()
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
				fmt.Printf("  ▸ [%s] %s\n", server, name)
				for k, v := range b.ToolCall.Value.Arguments {
					fmt.Printf("    %s: %v\n", k, v)
				}
			}
		case "toolResponse":
			if b.ToolResponse != nil && b.ToolResponse.Value != "" {
				val := b.ToolResponse.Value
				if len(val) > 500 {
					val = val[:500] + "..."
				}
				fmt.Printf("    → %s\n", val)
			}
		}
	}
}
