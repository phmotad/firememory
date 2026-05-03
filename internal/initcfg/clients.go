// Package initcfg writes MCP server entries into the configuration files used
// by popular AI coding clients (Claude Code, Cursor, Windsurf, …).
package initcfg

import (
	"os"
	"path/filepath"
	"runtime"
)

// Client describes how to locate a supported AI coding assistant's MCP config.
type Client struct {
	Name        string
	Description string
	configPath  func() string
}

// ConfigPath returns the absolute path to the client's MCP config file.
// Returns "" if the path cannot be determined.
func (c Client) ConfigPath() string {
	if c.configPath == nil {
		return ""
	}
	return c.configPath()
}

// Supported returns the list of clients this package knows how to configure.
func Supported() []Client {
	return []Client{
		claudeCode,
		cursor,
		windsurf,
		zed,
	}
}

// Find returns the Client matching name (case-insensitive prefix match).
func Find(name string) (Client, bool) {
	for _, c := range Supported() {
		if c.Name == name {
			return c, true
		}
	}
	return Client{}, false
}

var claudeCode = Client{
	Name:        "claude-code",
	Description: "Claude Code (Anthropic CLI)",
	configPath: func() string {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".claude.json")
	},
}

var cursor = Client{
	Name:        "cursor",
	Description: "Cursor IDE",
	configPath: func() string {
		home, _ := os.UserHomeDir()
		// Cursor on Windows uses AppData, on macOS/Linux uses ~/.cursor.
		if runtime.GOOS == "windows" {
			appData := os.Getenv("APPDATA")
			if appData == "" {
				appData = filepath.Join(home, "AppData", "Roaming")
			}
			return filepath.Join(appData, "Cursor", "User", "globalStorage", "cursor.mcp", "mcp.json")
		}
		return filepath.Join(home, ".cursor", "mcp.json")
	},
}

var windsurf = Client{
	Name:        "windsurf",
	Description: "Windsurf (Codeium)",
	configPath: func() string {
		home, _ := os.UserHomeDir()
		if runtime.GOOS == "windows" {
			appData := os.Getenv("APPDATA")
			if appData == "" {
				appData = filepath.Join(home, "AppData", "Roaming")
			}
			return filepath.Join(appData, "Codeium", "windsurf", "mcp_settings.json")
		}
		return filepath.Join(home, ".codeium", "windsurf", "mcp_settings.json")
	},
}

var zed = Client{
	Name:        "zed",
	Description: "Zed editor",
	configPath: func() string {
		home, _ := os.UserHomeDir()
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Application Support", "Zed", "settings.json")
		}
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, "zed", "settings.json")
	},
}
