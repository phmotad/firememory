package initcfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPatchCreatesNewFile(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "new.json")

	entry := MCPEntry{Command: "/usr/bin/fquery", Args: []string{"mcp"}}
	result, err := Patch(cfgPath, entry)
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if !result.Created {
		t.Error("expected Created=true for new file")
	}
	if result.Updated {
		t.Error("expected Updated=false for new file")
	}

	raw, _ := os.ReadFile(cfgPath)
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("written file is not valid JSON: %v", err)
	}
	servers := cfg["mcpServers"].(map[string]any)
	fq := servers["firequery"].(map[string]any)
	if fq["command"] != "/usr/bin/fquery" {
		t.Errorf("unexpected command: %v", fq["command"])
	}
}

func TestPatchUpdatesExistingEntry(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "existing.json")

	initial := `{"mcpServers":{"firequery":{"command":"/old/fquery","args":["mcp"]}}}`
	os.WriteFile(cfgPath, []byte(initial), 0o644)

	entry := MCPEntry{Command: "/new/fquery", Args: []string{"mcp"}}
	result, err := Patch(cfgPath, entry)
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if result.Created {
		t.Error("expected Created=false for existing file")
	}
	if !result.Updated {
		t.Error("expected Updated=true when replacing existing entry")
	}

	raw, _ := os.ReadFile(cfgPath)
	var cfg map[string]any
	json.Unmarshal(raw, &cfg)
	servers := cfg["mcpServers"].(map[string]any)
	fq := servers["firequery"].(map[string]any)
	if fq["command"] != "/new/fquery" {
		t.Errorf("command not updated, got: %v", fq["command"])
	}
}

func TestPatchPreservesOtherKeys(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "cfg.json")

	initial := `{"theme":"dark","mcpServers":{"other-tool":{"command":"other"}}}`
	os.WriteFile(cfgPath, []byte(initial), 0o644)

	Patch(cfgPath, MCPEntry{Command: "/fquery", Args: []string{"mcp"}})

	raw, _ := os.ReadFile(cfgPath)
	var cfg map[string]any
	json.Unmarshal(raw, &cfg)

	if cfg["theme"] != "dark" {
		t.Error("existing top-level key 'theme' was lost")
	}
	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["other-tool"]; !ok {
		t.Error("existing mcpServers entry 'other-tool' was lost")
	}
	if _, ok := servers["firequery"]; !ok {
		t.Error("firequery entry was not added")
	}
}
