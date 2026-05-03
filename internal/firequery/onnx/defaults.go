package onnx

import (
	"os"
	"path/filepath"
	"runtime"
)

const envModelsDir = "FIREMEMORY_MODELS_DIR"

// DefaultModelsDir returns the platform-appropriate default model cache directory.
// Override with the FIREMEMORY_MODELS_DIR environment variable.
func DefaultModelsDir() string {
	if v := os.Getenv(envModelsDir); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = os.Getenv("USERPROFILE")
		}
		return filepath.Join(base, "firememory", "models")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Caches", "firememory", "models")
	default:
		if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
			return filepath.Join(xdg, "firememory", "models")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".cache", "firememory", "models")
	}
}
