// Package defaultbrain resolves and auto-initialises the default brainfile
// (~/.firememory/default.fbrain) used when no explicit path is given.
package defaultbrain

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/phmotad/firememory/internal/brainfile"
)

const (
	dirName  = ".firememory"
	fileName = "default.fbrain"
)

// Path returns the absolute path to the default brainfile.
// It does not create anything.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("defaultbrain: home dir: %w", err)
	}
	return filepath.Join(home, dirName, fileName), nil
}

// EnsureExists creates the default brainfile if it does not already exist.
// Returns the absolute path.
func EnsureExists() (string, error) {
	p, err := Path()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(p); err == nil {
		return p, nil // already exists
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", fmt.Errorf("defaultbrain: mkdir: %w", err)
	}

	handle, err := brainfile.Create(p, brainfile.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("defaultbrain: create: %w", err)
	}
	defer handle.Close()

	return p, nil
}
