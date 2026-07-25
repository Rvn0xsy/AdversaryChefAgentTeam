package goose

import (
	"reflect"
	"testing"
)

func TestParsePromptMeta(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    PromptMeta
	}{
		{
			name: "echo-recon",
			content: `# AC-Echo
> **Purpose**: Map external attack surface
> **Requires**: nexus-mcp, kali-mcp
> **Skills**: red-team/kali
> **Input**: Target domain`,
			want: PromptMeta{
				Requires: []string{"nexus-mcp", "kali-mcp"},
				Skills:   []string{"red-team/kali"},
			},
		},
		{
			name: "quill-report - no skills",
			content: `# AC-Quill
> **Purpose**: Generate reports
> **Requires**: nexus-mcp
> **Skills**:
> **Input**: Project ID`,
			want: PromptMeta{
				Requires: []string{"nexus-mcp"},
				Skills:   nil,
			},
		},
		{
			name: "no fields",
			content: `# No metadata agent
Just does stuff.`,
			want: PromptMeta{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePromptMeta([]byte(tt.content))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePromptMeta() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
