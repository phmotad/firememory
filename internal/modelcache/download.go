package modelcache

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	downloadTimeout  = 30 * time.Minute
	userAgent        = "firememory-modelcache/1"
	tempSuffix       = ".download"
)

// downloadFile fetches url into destPath with resume support.
// It writes progress to w (nil = silent). Falls back to fallbackURL on failure.
func downloadFile(ctx context.Context, primaryURL, fallbackURL, destPath string, totalBytes int64, w io.Writer) error {
	tempPath := destPath + tempSuffix

	// Check existing partial download.
	var offset int64
	if fi, err := os.Stat(tempPath); err == nil {
		offset = fi.Size()
	}

	err := fetchWithResume(ctx, primaryURL, tempPath, offset, totalBytes, w)
	if err != nil && fallbackURL != "" && fallbackURL != primaryURL {
		// Log fallback and retry from scratch.
		if w != nil {
			fmt.Fprintf(w, "    primary failed (%v), trying fallback...\n", err)
		}
		_ = os.Remove(tempPath)
		err = fetchWithResume(ctx, fallbackURL, tempPath, 0, totalBytes, w)
	}
	if err != nil {
		return err
	}

	return os.Rename(tempPath, destPath)
}

func fetchWithResume(ctx context.Context, url, destPath string, offset, totalBytes int64, w io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", shortenURL(url), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		// Server doesn't support resume or file is fully downloaded.
		offset = 0
		resp.Body.Close()
		req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		req2.Header.Set("User-Agent", userAgent)
		resp2, err := client.Do(req2)
		if err != nil {
			return fmt.Errorf("retry GET %s: %w", shortenURL(url), err)
		}
		defer resp2.Body.Close()
		return writeWithProgress(resp2.Body, destPath, 0, totalBytes, w)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("GET %s: HTTP %d", shortenURL(url), resp.StatusCode)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if offset > 0 && resp.StatusCode == http.StatusPartialContent {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
		offset = 0
	}

	return writeWithProgress(resp.Body, destPath, offset, totalBytes, w)
}

func writeWithProgress(body io.Reader, destPath string, alreadyBytes, totalBytes int64, w io.Writer) error {
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open dest: %w", err)
	}
	defer f.Close()

	pr := &progressReader{
		r:     body,
		written: alreadyBytes,
		total:  totalBytes,
		w:      w,
		last:   time.Now(),
	}
	_, err = io.Copy(f, pr)
	if w != nil {
		fmt.Fprintln(w) // newline after progress bar
	}
	return err
}

// progressReader wraps a reader and prints a simple progress line to w.
type progressReader struct {
	r       io.Reader
	written int64
	total   int64
	w       io.Writer
	last    time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.written += int64(n)
	if p.w != nil && time.Since(p.last) > 200*time.Millisecond {
		p.printProgress()
		p.last = time.Now()
	}
	return n, err
}

func (p *progressReader) printProgress() {
	if p.total <= 0 {
		fmt.Fprintf(p.w, "\r    %.1f MB downloaded...", float64(p.written)/1e6)
		return
	}
	pct := float64(p.written) / float64(p.total) * 100
	bar := int(pct / 5)
	filled := strings.Repeat("█", bar)
	empty := strings.Repeat("░", 20-bar)
	fmt.Fprintf(p.w, "\r    [%s%s] %.0f%% (%.1f/%.1f MB)",
		filled, empty, pct,
		float64(p.written)/1e6,
		float64(p.total)/1e6,
	)
}

func shortenURL(url string) string {
	if len(url) > 60 {
		return url[:57] + "..."
	}
	return url
}
