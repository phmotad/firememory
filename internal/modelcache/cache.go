package modelcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ModelStatus reports the cached state of one model.
type ModelStatus struct {
	ID       string
	Label    string
	Dir      string
	Present  bool
	Verified bool // SHA256 matched (or skipped for placeholders)
	Error    string
}

// Status returns the cache status of every model in the manifest.
func Status(cacheDir string) ([]ModelStatus, error) {
	manifest, err := Load()
	if err != nil {
		return nil, err
	}
	out := make([]ModelStatus, 0, len(manifest.Models))
	for _, m := range manifest.Models {
		out = append(out, checkModel(cacheDir, m))
	}
	return out, nil
}

func checkModel(cacheDir string, m ModelEntry) ModelStatus {
	s := ModelStatus{ID: m.ID, Label: m.Label, Dir: filepath.Join(cacheDir, m.Dir)}

	onnxPath := filepath.Join(cacheDir, m.Dir, "model.onnx")
	tokPath := filepath.Join(cacheDir, m.Dir, "tokenizer.json")

	for _, p := range []string{onnxPath, tokPath} {
		if _, err := os.Stat(p); err != nil {
			s.Error = "missing " + filepath.Base(p)
			return s
		}
	}
	s.Present = true

	if m.IsPlaceholder() {
		s.Verified = true // skip hash check for dev placeholders
		return s
	}

	hashPath := filepath.Join(cacheDir, m.Dir, ".sha256")
	stored, err := os.ReadFile(hashPath)
	if err != nil {
		s.Error = "checksum file missing"
		return s
	}
	if string(stored) != m.SHA256 {
		s.Error = "checksum mismatch"
		return s
	}
	s.Verified = true
	return s
}

// AllPresent returns true when every model in the manifest is present and verified.
func AllPresent(cacheDir string) (bool, error) {
	statuses, err := Status(cacheDir)
	if err != nil {
		return false, err
	}
	for _, s := range statuses {
		if !s.Present || !s.Verified {
			return false, nil
		}
	}
	return true, nil
}

// verifyArchive computes the SHA256 of a file and compares it to expected.
func verifyArchive(path, expected string) error {
	if expected == "" || expected == "PLACEHOLDER" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open for verify: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash read: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		return fmt.Errorf("SHA256 mismatch: got %s want %s", got[:16]+"...", expected[:16]+"...")
	}
	return nil
}

// writeChecksum saves the SHA256 of an archive to the model directory.
func writeChecksum(modelDir, sha256sum string) error {
	if sha256sum == "" || sha256sum == "PLACEHOLDER" {
		return nil
	}
	return os.WriteFile(filepath.Join(modelDir, ".sha256"), []byte(sha256sum), 0o644)
}
