package goose

import (
	"regexp"
	"strings"
)

// PromptMeta holds parsed agent prompt header fields.
type PromptMeta struct {
	Requires []string // MCP names from "> **Requires**: ..."
	Skills   []string // Skill paths from "> **Skills**: ..."
}

var (
	reRequires = regexp.MustCompile(`\*{1,2}Requires\*{1,2}:[ \t]*(.*)`)
	reSkills   = regexp.MustCompile(`\*{1,2}Skills\*{1,2}:[ \t]*(.*)`)
)

// ParsePromptMeta extracts Requires and Skills from a prompt file's header block.
// Handles the format: > **Requires**: nexus-mcp, kali-mcp
func ParsePromptMeta(content []byte) PromptMeta {
	meta := PromptMeta{}
	text := string(content)

	if m := reRequires.FindStringSubmatch(text); len(m) >= 2 {
		meta.Requires = splitTrim(m[1])
	}
	if m := reSkills.FindStringSubmatch(text); len(m) >= 2 {
		meta.Skills = splitTrim(m[1])
	}
	return meta
}

func splitTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
