package brainfile

import (
	"errors"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phmotad/firememory/internal/storage"
	bolt "go.etcd.io/bbolt"
)

func TestValidatePathRequiresFbrainExtension(t *testing.T) {
	if err := ValidatePath("agent.txt"); err != ErrInvalidExtension {
		t.Fatalf("expected ErrInvalidExtension, got %v", err)
	}
}

func TestCreateInitializesManifestAndNamespaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	handle, err := Create(path, CreateOptions{Name: "agent"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer handle.Close()

	manifest := handle.Manifest()
	if manifest.Name != "agent" {
		t.Fatalf("expected manifest name %q, got %q", "agent", manifest.Name)
	}

	if manifest.Extension != Extension {
		t.Fatalf("expected extension %q, got %q", Extension, manifest.Extension)
	}

	if manifest.FormatVersion != FormatVersion {
		t.Fatalf("expected format version %q, got %q", FormatVersion, manifest.FormatVersion)
	}

	snapshot, err := handle.Store().Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	for _, namespace := range OfficialNamespaces() {
		if _, ok := snapshot.Namespaces[namespace]; !ok {
			t.Fatalf("expected namespace %q to exist", namespace)
		}
	}
}

func TestOpenLoadsPersistedManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "support-agent.fbrain")

	created, err := Create(path, CreateOptions{
		Name:           "support-agent",
		Version:        "0.2.0",
		EmbeddingModel: "deterministic",
		EmbeddingDim:   128,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	expected := created.Manifest()
	if err := created.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	opened, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer opened.Close()

	manifest := opened.Manifest()
	if manifest.ID != expected.ID {
		t.Fatalf("expected id %q, got %q", expected.ID, manifest.ID)
	}

	if manifest.Name != expected.Name {
		t.Fatalf("expected name %q, got %q", expected.Name, manifest.Name)
	}

	if manifest.EmbeddingDim != expected.EmbeddingDim {
		t.Fatalf("expected embedding dim %d, got %d", expected.EmbeddingDim, manifest.EmbeddingDim)
	}
}

func TestInspectReturnsNamespaceCounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "company-memory.fbrain")

	handle, err := Create(path, CreateOptions{Name: "company-memory"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := handle.Store().Put("memories", "mem_01", []byte("alpha")); err != nil {
		t.Fatalf("put memory: %v", err)
	}

	if err := handle.Store().Put("facts", "fact_01", []byte("beta")); err != nil {
		t.Fatalf("put fact: %v", err)
	}

	if err := handle.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	inspection, err := Inspect(path)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}

	if inspection.Manifest.Name != "company-memory" {
		t.Fatalf("expected manifest name %q, got %q", "company-memory", inspection.Manifest.Name)
	}

	if inspection.NamespaceCounts["memories"] != 1 {
		t.Fatalf("expected 1 memory record, got %d", inspection.NamespaceCounts["memories"])
	}

	if inspection.NamespaceCounts["facts"] != 1 {
		t.Fatalf("expected 1 fact record, got %d", inspection.NamespaceCounts["facts"])
	}

	if inspection.NamespaceCounts[ManifestNamespace] != 1 {
		t.Fatalf("expected 1 manifest record, got %d", inspection.NamespaceCounts[ManifestNamespace])
	}
}

func TestOpenWithoutManifestFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.fbrain")

	store, err := storage.OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open raw store: %v", err)
	}

	if err := store.EnsureNamespace("memories"); err != nil {
		t.Fatalf("ensure namespace: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close raw store: %v", err)
	}

	_, err = Open(path)
	if err != ErrManifestNotFound {
		t.Fatalf("expected ErrManifestNotFound, got %v", err)
	}
}

func TestBackupAndRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	handle, err := Create(path, CreateOptions{Name: "agent"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := handle.Store().Put("memories", "mem_01", []byte("alpha")); err != nil {
		t.Fatalf("put memory: %v", err)
	}

	if err := handle.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "agent.backup")
	if err := Backup(path, backupPath); err != nil {
		t.Fatalf("backup: %v", err)
	}

	restoredPath := filepath.Join(t.TempDir(), "restored-agent.fbrain")
	if err := Restore(backupPath, restoredPath); err != nil {
		t.Fatalf("restore: %v", err)
	}

	restored, err := Open(restoredPath)
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}
	defer restored.Close()

	value, err := restored.Store().Get("memories", "mem_01")
	if err != nil {
		t.Fatalf("get restored memory: %v", err)
	}
	if string(value) != "alpha" {
		t.Fatalf("expected restored value %q, got %q", "alpha", string(value))
	}
}

func TestOpenFailsIntegrityValidationWhenNamespaceIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken-integrity.fbrain")

	handle, err := Create(path, CreateOptions{Name: "broken"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "broken-integrity.backup")
	if err := Backup(path, backupPath); err != nil {
		t.Fatalf("backup: %v", err)
	}

	db, err := bolt.Open(path, 0o600, nil)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("vectors"))
	})
	if err != nil {
		t.Fatalf("delete bucket: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	_, err = Open(path)
	if err == nil || !errors.Is(err, ErrIntegrityViolation) {
		t.Fatalf("expected ErrIntegrityViolation, got %v", err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove broken file: %v", err)
	}
	if err := Restore(backupPath, path); err != nil {
		t.Fatalf("restore after corruption: %v", err)
	}
}

func TestOpenMigratesLegacyFormatVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.fbrain")

	store, err := storage.OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	legacyManifest := Manifest{
		ID:             "brain_legacy",
		Name:           "legacy-agent",
		Version:        "0.0.5",
		FormatVersion:  LegacyFormatVersion,
		Extension:      Extension,
		EmbeddingModel: "",
		EmbeddingDim:   0,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	payload, err := json.Marshal(legacyManifest)
	if err != nil {
		t.Fatalf("marshal legacy manifest: %v", err)
	}
	if err := store.EnsureNamespace(ManifestNamespace); err != nil {
		t.Fatalf("ensure manifest namespace: %v", err)
	}
	if err := store.Put(ManifestNamespace, ManifestKey, payload); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := store.EnsureNamespace("memories"); err != nil {
		t.Fatalf("ensure memories namespace: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	handle, err := Open(path)
	if err != nil {
		t.Fatalf("open migrated brainfile: %v", err)
	}
	defer handle.Close()

	manifest := handle.Manifest()
	if manifest.FormatVersion != FormatVersion {
		t.Fatalf("format_version = %q, want %q", manifest.FormatVersion, FormatVersion)
	}
	if manifest.EmbeddingModel != DefaultEmbedder {
		t.Fatalf("embedding_model = %q, want %q", manifest.EmbeddingModel, DefaultEmbedder)
	}
	if manifest.EmbeddingDim != DefaultEmbedDim {
		t.Fatalf("embedding_dim = %d, want %d", manifest.EmbeddingDim, DefaultEmbedDim)
	}

	snapshot, err := handle.Store().Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	for _, namespace := range OfficialNamespaces() {
		if _, ok := snapshot.Namespaces[namespace]; !ok {
			t.Fatalf("expected namespace %q after migration", namespace)
		}
	}
}

func TestOpenRejectsUnsupportedFormatVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.fbrain")

	store, err := storage.OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	manifest := Manifest{
		ID:             "brain_future",
		Name:           "future-agent",
		Version:        "9.9.0",
		FormatVersion:  "9.9",
		Extension:      Extension,
		EmbeddingModel: DefaultEmbedder,
		EmbeddingDim:   DefaultEmbedDim,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	for _, namespace := range OfficialNamespaces() {
		if err := store.EnsureNamespace(namespace); err != nil {
			t.Fatalf("ensure namespace %q: %v", namespace, err)
		}
	}
	if err := store.Put(ManifestNamespace, ManifestKey, payload); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	_, err = Open(path)
	if err == nil || !errors.Is(err, ErrUnsupportedFormatVersion) {
		t.Fatalf("expected ErrUnsupportedFormatVersion, got %v", err)
	}
}
