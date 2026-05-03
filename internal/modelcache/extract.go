package modelcache

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractTarGz extracts a .tar.gz archive into destDir.
// Only regular files are extracted; directory entries and symlinks are skipped.
func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		// Strip any leading directory component so files land flat in destDir.
		name := filepath.Base(hdr.Name)
		if name == "" || name == "." {
			continue
		}

		destPath := filepath.Join(destDir, name)
		// Path traversal guard.
		if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("tar: invalid path %q", hdr.Name)
		}

		if err := writeFile(tr, destPath, hdr.Mode); err != nil {
			return fmt.Errorf("extract %s: %w", name, err)
		}
	}
	return nil
}

func writeFile(r io.Reader, destPath string, mode int64) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(mode)|0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}
