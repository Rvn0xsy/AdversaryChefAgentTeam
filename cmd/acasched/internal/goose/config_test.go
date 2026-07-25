package goose

import (
	"os"
	"testing"
)

func TestLoadRegistry(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/registry.yaml"
	os.WriteFile(path, []byte("nexus-mcp: http://127.0.0.1:8081\nkali-mcp: http://127.0.0.1:8080\n"), 0644)

	reg, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if reg["nexus-mcp"] != "http://127.0.0.1:8081" {
		t.Errorf("got nexus-mcp=%q", reg["nexus-mcp"])
	}
	if reg["kali-mcp"] != "http://127.0.0.1:8080" {
		t.Errorf("got kali-mcp=%q", reg["kali-mcp"])
	}
}

func TestLoadRegistryMissing(t *testing.T) {
	_, err := LoadRegistry("/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadSquads(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/squads.yaml"
	os.WriteFile(path, []byte(`squads:
  red-team:
    description: "Red Team"
    prompt_dir: "red-team"
    skill_dir: "red-team"
`), 0644)

	squads, err := LoadSquads(path)
	if err != nil {
		t.Fatalf("LoadSquads: %v", err)
	}
	if _, ok := squads["red-team"]; !ok {
		t.Error("missing red-team squad")
	}
	if squads["red-team"].PromptDir != "red-team" {
		t.Errorf("prompt_dir=%q", squads["red-team"].PromptDir)
	}
}
