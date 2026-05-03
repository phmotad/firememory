package brainfile

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/phmotad/firememory/internal/storage"
)

const LegacyFormatVersion = "0.0"

var (
	ErrUnsupportedFormatVersion = errors.New("unsupported format version")
	ErrNoMigrationPath          = errors.New("no migration path for format version")
)

type Migration struct {
	From  string
	To    string
	Apply func(store storage.Store, manifest Manifest) (Manifest, error)
}

func SupportedFormatVersions() []string {
	return []string{LegacyFormatVersion, FormatVersion}
}

func CanOpenFormatVersion(version string) bool {
	for _, supported := range SupportedFormatVersions() {
		if version == supported {
			return true
		}
	}
	return false
}

func Migrate(store storage.Store, manifest Manifest) (Manifest, error) {
	current := strings.TrimSpace(manifest.FormatVersion)
	if current == "" {
		return Manifest{}, ErrFormatVersionEmpty
	}

	if current == FormatVersion {
		return manifest, nil
	}
	if !CanOpenFormatVersion(current) {
		return Manifest{}, fmt.Errorf("%w: %s", ErrUnsupportedFormatVersion, current)
	}

	migrations := registeredMigrations()
	for current != FormatVersion {
		migration, ok := migrations[current]
		if !ok {
			return Manifest{}, fmt.Errorf("%w: %s", ErrNoMigrationPath, current)
		}

		nextManifest, err := migration.Apply(store, manifest)
		if err != nil {
			return Manifest{}, err
		}
		manifest = nextManifest
		current = manifest.FormatVersion
	}

	if err := writeManifest(store, manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func registeredMigrations() map[string]Migration {
	return map[string]Migration{
		LegacyFormatVersion: {
			From: LegacyFormatVersion,
			To:   FormatVersion,
			Apply: func(store storage.Store, manifest Manifest) (Manifest, error) {
				for _, namespace := range officialNamespaces {
					if err := store.EnsureNamespace(namespace); err != nil {
						return Manifest{}, err
					}
				}

				migrated := manifest
				if strings.TrimSpace(migrated.ID) == "" {
					migrated.ID = "brain_" + time.Now().UTC().Format("20060102150405.000000000")
				}
				if strings.TrimSpace(migrated.Name) == "" {
					migrated.Name = "agent"
				}
				if strings.TrimSpace(migrated.Version) == "" {
					migrated.Version = DefaultVersion
				}
				if strings.TrimSpace(migrated.Extension) == "" {
					migrated.Extension = Extension
				}
				if strings.TrimSpace(migrated.EmbeddingModel) == "" {
					migrated.EmbeddingModel = DefaultEmbedder
				}
				if migrated.EmbeddingDim == 0 {
					migrated.EmbeddingDim = DefaultEmbedDim
				}
				if migrated.CreatedAt.IsZero() {
					migrated.CreatedAt = time.Now().UTC()
				}
				migrated.UpdatedAt = time.Now().UTC()
				migrated.FormatVersion = FormatVersion
				return migrated, nil
			},
		},
	}
}

func requiresMigration(version string) bool {
	return version != "" && version != FormatVersion && slices.Contains(SupportedFormatVersions(), version)
}
