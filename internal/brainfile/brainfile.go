package brainfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/phmotad/firememory/internal/storage"
)

const (
	Extension         = ".fbrain"
	FormatVersion     = "0.1"
	DefaultVersion    = "0.1.0"
	DefaultEmbedder   = "deterministic"
	DefaultEmbedDim   = 384
	ManifestNamespace = "manifest"
	ManifestKey       = "brain_manifest"
)

var (
	ErrInvalidExtension   = errors.New("brainfile path must end with .fbrain")
	ErrManifestNotFound   = errors.New("brain manifest not found")
	ErrInvalidManifest    = errors.New("invalid brain manifest")
	ErrFormatVersionEmpty = errors.New("format version is required")
	ErrInvalidBackupPath  = errors.New("backup path is required")
	ErrIntegrityViolation = errors.New("brainfile integrity check failed")
)

var officialNamespaces = []string{
	ManifestNamespace,
	"memories",
	"entities",
	"relations",
	"facts",
	"events",
	"concepts",
	"sources",
	"vectors",
	"graph_nodes",
	"graph_edges",
	"hash_index",
	"traces",
	"sync_queue",
}

type Manifest struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Version        string    `json:"version"`
	FormatVersion  string    `json:"format_version"`
	Extension      string    `json:"extension"`
	EmbeddingModel string    `json:"embedding_model"`
	EmbeddingDim   int       `json:"embedding_dim"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateOptions struct {
	ID             string
	Name           string
	Version        string
	EmbeddingModel string
	EmbeddingDim   int
}

type Handle struct {
	path     string
	store    storage.Store
	manifest Manifest
}

type Inspection struct {
	Path            string
	Manifest        Manifest
	Namespaces      []string
	NamespaceCounts map[string]int
}

func OfficialNamespaces() []string {
	return slices.Clone(officialNamespaces)
}

func ValidatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return storage.ErrPathRequired
	}

	if !strings.HasSuffix(strings.ToLower(path), Extension) {
		return ErrInvalidExtension
	}

	return nil
}

func Create(path string, opts CreateOptions) (*Handle, error) {
	if err := ValidatePath(path); err != nil {
		return nil, err
	}

	store, err := storage.OpenBboltStore(path)
	if err != nil {
		return nil, err
	}

	handle := &Handle{
		path:  path,
		store: store,
	}

	manifest := defaultManifest(path, opts)
	if err := manifest.Validate(); err != nil {
		_ = store.Close()
		return nil, err
	}

	if err := initializeNamespaces(store); err != nil {
		_ = store.Close()
		return nil, err
	}

	if err := writeManifest(store, manifest); err != nil {
		_ = store.Close()
		return nil, err
	}

	handle.manifest = manifest
	return handle, nil
}

func Open(path string) (*Handle, error) {
	if err := ValidatePath(path); err != nil {
		return nil, err
	}

	store, err := storage.OpenBboltStore(path)
	if err != nil {
		return nil, err
	}

	manifest, err := readManifest(store)
	if err != nil {
		_ = store.Close()
		return nil, err
	}

	if requiresMigration(manifest.FormatVersion) {
		manifest, err = Migrate(store, manifest)
		if err != nil {
			_ = store.Close()
			return nil, err
		}
	}

	if err := validateIntegrity(store, manifest); err != nil {
		_ = store.Close()
		return nil, err
	}

	if err := initializeNamespaces(store); err != nil {
		_ = store.Close()
		return nil, err
	}

	return &Handle{
		path:     path,
		store:    store,
		manifest: manifest,
	}, nil
}

func Inspect(path string) (Inspection, error) {
	handle, err := Open(path)
	if err != nil {
		return Inspection{}, err
	}
	defer handle.Close()

	snapshot, err := handle.store.Snapshot()
	if err != nil {
		return Inspection{}, err
	}

	namespaces := make([]string, 0, len(snapshot.Namespaces))
	counts := make(map[string]int, len(snapshot.Namespaces))
	for _, namespace := range officialNamespaces {
		records := snapshot.Namespaces[namespace]
		namespaces = append(namespaces, namespace)
		counts[namespace] = len(records)
	}

	return Inspection{
		Path:            path,
		Manifest:        handle.manifest,
		Namespaces:      namespaces,
		NamespaceCounts: counts,
	}, nil
}

func (h *Handle) Path() string {
	return h.path
}

func (h *Handle) Manifest() Manifest {
	return h.manifest
}

func (h *Handle) Store() storage.Store {
	return h.store
}

func (h *Handle) Close() error {
	return h.store.Close()
}

func Backup(sourcePath, backupPath string) error {
	if err := ValidatePath(sourcePath); err != nil {
		return err
	}
	if strings.TrimSpace(backupPath) == "" {
		return ErrInvalidBackupPath
	}

	handle, err := Open(sourcePath)
	if err != nil {
		return err
	}
	if err := handle.Close(); err != nil {
		return err
	}

	return copyFile(sourcePath, backupPath)
}

func Restore(backupPath, destinationPath string) error {
	if strings.TrimSpace(backupPath) == "" {
		return ErrInvalidBackupPath
	}
	if err := ValidatePath(destinationPath); err != nil {
		return err
	}

	tempPath := destinationPath + ".restore.tmp.fbrain"
	if err := copyFile(backupPath, tempPath); err != nil {
		return err
	}

	handle, err := Open(tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := handle.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	if err := os.Rename(tempPath, destinationPath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	return nil
}

func (m Manifest) Validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return ErrInvalidManifest
	}

	if strings.TrimSpace(m.Name) == "" {
		return ErrInvalidManifest
	}

	if strings.TrimSpace(m.Version) == "" {
		return ErrInvalidManifest
	}

	if strings.TrimSpace(m.FormatVersion) == "" {
		return ErrFormatVersionEmpty
	}

	if strings.TrimSpace(m.Extension) == "" {
		return ErrInvalidExtension
	}

	if m.Extension != Extension {
		return ErrInvalidExtension
	}

	if m.EmbeddingDim < 0 {
		return ErrInvalidManifest
	}

	return nil
}

func defaultManifest(path string, opts CreateOptions) Manifest {
	now := time.Now().UTC()
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), Extension)
	}

	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = DefaultVersion
	}

	embeddingModel := strings.TrimSpace(opts.EmbeddingModel)
	if embeddingModel == "" {
		embeddingModel = DefaultEmbedder
	}

	embeddingDim := opts.EmbeddingDim
	if embeddingDim == 0 {
		embeddingDim = DefaultEmbedDim
	}

	id := strings.TrimSpace(opts.ID)
	if id == "" {
		id = "brain_" + now.Format("20060102150405.000000000")
	}

	return Manifest{
		ID:             id,
		Name:           name,
		Version:        version,
		FormatVersion:  FormatVersion,
		Extension:      Extension,
		EmbeddingModel: embeddingModel,
		EmbeddingDim:   embeddingDim,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func initializeNamespaces(store storage.Store) error {
	for _, namespace := range officialNamespaces {
		if err := store.EnsureNamespace(namespace); err != nil {
			return err
		}
	}

	return nil
}

func writeManifest(store storage.Store, manifest Manifest) error {
	payload, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	return store.Put(ManifestNamespace, ManifestKey, payload)
}

func readManifest(store storage.Store) (Manifest, error) {
	payload, err := store.Get(ManifestNamespace, ManifestKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return Manifest{}, ErrManifestNotFound
		}

		return Manifest{}, err
	}

	var manifest Manifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return Manifest{}, err
	}

	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func validateIntegrity(store storage.Store, manifest Manifest) error {
	if !CanOpenFormatVersion(manifest.FormatVersion) {
		return fmt.Errorf("%w: %w %q", ErrIntegrityViolation, ErrUnsupportedFormatVersion, manifest.FormatVersion)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		return err
	}

	for _, namespace := range officialNamespaces {
		if _, ok := snapshot.Namespaces[namespace]; !ok {
			return fmt.Errorf("%w: missing namespace %q", ErrIntegrityViolation, namespace)
		}
	}

	if len(snapshot.Namespaces[ManifestNamespace]) == 0 {
		return fmt.Errorf("%w: manifest namespace is empty", ErrIntegrityViolation)
	}

	return nil
}

func copyFile(sourcePath, destinationPath string) error {
	dir := filepath.Dir(destinationPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(destinationPath)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	return nil
}
