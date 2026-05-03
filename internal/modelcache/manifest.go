// Package modelcache manages automatic download and local caching of ONNX models
// required by the FireQuery inference pipeline.
//
// Models are downloaded from GitHub Releases (primary) with a HuggingFace fallback,
// extracted to a per-OS cache directory, and verified by SHA256 checksum.
// All progress output goes to stderr so it is safe alongside MCP stdio transport.
package modelcache

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed manifest.json
var manifestJSON []byte

// ModelEntry describes one downloadable model archive.
type ModelEntry struct {
	ID              string `json:"id"`
	Version         string `json:"version"`
	Dir             string `json:"dir"`
	Archive         string `json:"archive"`
	PrimaryURL      string `json:"primary_url"`
	FallbackURL     string `json:"fallback_url"`
	SHA256          string `json:"sha256"`
	CompressedBytes int64  `json:"compressed_bytes"`
	Label           string `json:"label"`
}

// IsPlaceholder reports whether the SHA256 is a development placeholder.
// Verification is skipped for placeholders (pre-release development only).
func (m ModelEntry) IsPlaceholder() bool {
	return m.SHA256 == "" || m.SHA256 == "PLACEHOLDER" || len(m.SHA256) < 16
}

// Manifest is the embedded list of downloadable models.
type Manifest struct {
	SchemaVersion string       `json:"schema_version"`
	ModelsTag     string       `json:"models_tag"`
	Models        []ModelEntry `json:"models"`
}

// Load parses the manifest embedded in the binary.
func Load() (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return nil, fmt.Errorf("modelcache: parse manifest: %w", err)
	}
	return &m, nil
}
