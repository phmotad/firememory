package modelcache

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EnsureAll downloads any missing or unverified models in the manifest.
// Already-present and verified models are skipped. Progress is written to w (nil = silent).
// This is the function called automatically on first use of fquery mcp / fmem sync.
func EnsureAll(ctx context.Context, cacheDir string, w io.Writer) error {
	manifest, err := Load()
	if err != nil {
		return err
	}

	var missing []ModelEntry
	for _, m := range manifest.Models {
		s := checkModel(cacheDir, m)
		if !s.Present || !s.Verified {
			missing = append(missing, m)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	totalMB := int64(0)
	for _, m := range missing {
		totalMB += m.CompressedBytes / 1_000_000
	}

	if w != nil {
		fmt.Fprintf(w, "\n[FireMemory] First run: downloading %d model(s) (~%d MB) to %s\n", len(missing), totalMB, cacheDir)
		fmt.Fprintf(w, "[FireMemory] This runs once. Subsequent starts are instant.\n\n")
	}

	for i, m := range missing {
		if err := pullOne(ctx, cacheDir, m, i+1, len(missing), w); err != nil {
			return fmt.Errorf("modelcache: download %s: %w", m.ID, err)
		}
	}

	if w != nil {
		fmt.Fprintf(w, "\n[FireMemory] Models ready.\n\n")
	}
	return nil
}

// PullAll force-downloads all models, replacing any existing files.
func PullAll(ctx context.Context, cacheDir string, w io.Writer) error {
	manifest, err := Load()
	if err != nil {
		return err
	}
	for i, m := range manifest.Models {
		if err := pullOne(ctx, cacheDir, m, i+1, len(manifest.Models), w); err != nil {
			return fmt.Errorf("modelcache: pull %s: %w", m.ID, err)
		}
	}
	if w != nil {
		fmt.Fprintf(w, "\n[FireMemory] All models downloaded.\n")
	}
	return nil
}

func pullOne(ctx context.Context, cacheDir string, m ModelEntry, idx, total int, w io.Writer) error {
	modelDir := filepath.Join(cacheDir, m.Dir)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	archivePath := filepath.Join(cacheDir, m.Archive)

	if w != nil {
		fmt.Fprintf(w, "  [%d/%d] %s (%s)\n", idx, total, m.Label, formatBytes(m.CompressedBytes))
	}

	if err := downloadFile(ctx, m.PrimaryURL, m.FallbackURL, archivePath, m.CompressedBytes, w); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	if err := verifyArchive(archivePath, m.SHA256); err != nil {
		_ = os.Remove(archivePath)
		return fmt.Errorf("verify: %w", err)
	}

	if w != nil {
		fmt.Fprintf(w, "    extracting...\n")
	}
	if err := extractTarGz(archivePath, modelDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	// Remove archive after successful extraction.
	_ = os.Remove(archivePath)

	if err := writeChecksum(modelDir, m.SHA256); err != nil {
		return fmt.Errorf("write checksum: %w", err)
	}

	if w != nil {
		fmt.Fprintf(w, "    done.\n")
	}
	return nil
}

// Remove deletes all downloaded model files from cacheDir.
func Remove(cacheDir string) error {
	manifest, err := Load()
	if err != nil {
		return err
	}
	for _, m := range manifest.Models {
		dir := filepath.Join(cacheDir, m.Dir)
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove %s: %w", m.ID, err)
		}
	}
	return nil
}

func formatBytes(b int64) string {
	if b >= 1_000_000_000 {
		return fmt.Sprintf("%.1f GB", float64(b)/1e9)
	}
	return fmt.Sprintf("%.0f MB", float64(b)/1e6)
}
