package initcfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MCPEntry is the MCP server entry written into agent config files.
type MCPEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// PatchResult describes what was done.
type PatchResult struct {
	ConfigPath string
	Created    bool // true if the config file did not exist before
	Updated    bool // true if an existing entry was replaced
}

// Patch reads (or creates) the client config at configPath and injects or
// updates the mcpServers."firequery" entry, then writes it back.
// The file is created (with parent dirs) if it does not exist.
func Patch(configPath string, entry MCPEntry) (PatchResult, error) {
	result := PatchResult{ConfigPath: configPath}

	raw, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		raw = []byte("{}")
		result.Created = true
	} else if err != nil {
		return result, fmt.Errorf("read config: %w", err)
	}

	// Unmarshal into a generic map so we preserve all existing keys.
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return result, fmt.Errorf("parse config: not valid JSON (%w)", err)
	}

	// Read or create the mcpServers object.
	var servers map[string]json.RawMessage
	if raw, ok := cfg["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return result, fmt.Errorf("parse mcpServers: %w", err)
		}
	} else {
		servers = make(map[string]json.RawMessage)
	}

	// Check whether we are replacing an existing entry.
	_, result.Updated = servers["firequery"]
	if result.Created {
		result.Updated = false
	}

	// Marshal the new entry.
	entryBytes, err := json.Marshal(entry)
	if err != nil {
		return result, fmt.Errorf("marshal entry: %w", err)
	}
	servers["firequery"] = entryBytes

	// Write the mcpServers key back.
	serversBytes, err := json.Marshal(servers)
	if err != nil {
		return result, fmt.Errorf("marshal mcpServers: %w", err)
	}
	cfg["mcpServers"] = serversBytes

	// Marshal the final config with 2-space indent to keep it human-readable.
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return result, fmt.Errorf("marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return result, fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(configPath, append(out, '\n'), 0o644); err != nil {
		return result, fmt.Errorf("write config: %w", err)
	}

	return result, nil
}

// Print returns the JSON that Patch would write, without touching the file.
func Print(configPath string, entry MCPEntry) ([]byte, error) {
	raw, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		raw = []byte("{}")
	} else if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	var servers map[string]json.RawMessage
	if rawServers, ok := cfg["mcpServers"]; ok {
		_ = json.Unmarshal(rawServers, &servers)
	}
	if servers == nil {
		servers = make(map[string]json.RawMessage)
	}

	entryBytes, _ := json.Marshal(entry)
	servers["firequery"] = entryBytes
	serversBytes, _ := json.Marshal(servers)
	cfg["mcpServers"] = serversBytes

	return json.MarshalIndent(cfg, "", "  ")
}
