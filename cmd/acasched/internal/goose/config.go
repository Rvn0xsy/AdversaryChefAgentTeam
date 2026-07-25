package goose

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SquadConfig holds a squad's directory mapping from _squads.yaml.
type SquadConfig struct {
	Description string `yaml:"description"`
	PromptDir   string `yaml:"prompt_dir"`
	SkillDir    string `yaml:"skill_dir"`
}

type squadFile struct {
	Squads map[string]SquadConfig `yaml:"squads"`
}

// LoadRegistry reads _mcp-registry.yaml and returns MCP name → URL map.
func LoadRegistry(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	registry := map[string]string{}
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return registry, nil
}

// LoadSquads reads _squads.yaml and returns squad name → SquadConfig map.
func LoadSquads(path string) (map[string]SquadConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read squads: %w", err)
	}
	var sf squadFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parse squads: %w", err)
	}
	return sf.Squads, nil
}
