package memory

import (
	"errors"
	"strings"
	"time"
)

const DefaultScope = "default"

var (
	ErrInvalidMemoryKind   = errors.New("invalid memory kind")
	ErrInvalidMemoryStatus = errors.New("invalid memory status")
	ErrInvalidDedupAction  = errors.New("invalid dedup action")
	ErrInvalidRelationType = errors.New("invalid relation type")
	ErrInvalidConfidence   = errors.New("confidence must be between 0 and 1")
	ErrInvalidImportance   = errors.New("importance must be between 0 and 1")
	ErrEmptyContent        = errors.New("content is required")
	ErrEmptyName           = errors.New("name is required")
	ErrEmptyRelationSide   = errors.New("relation endpoints are required")
	ErrEmptyFactField      = errors.New("fact subject, predicate, and object are required")
	ErrEmptySourceRef      = errors.New("source reference must include id, uri, or title")
)

type MemoryKind string

const (
	MemoryKindNote    MemoryKind = "note"
	MemoryKindFact    MemoryKind = "fact"
	MemoryKindEvent   MemoryKind = "event"
	MemoryKindConcept MemoryKind = "concept"
)

func (k MemoryKind) Valid() bool {
	switch k {
	case MemoryKindNote, MemoryKindFact, MemoryKindEvent, MemoryKindConcept:
		return true
	default:
		return false
	}
}

type MemoryStatus string

const (
	MemoryStatusActive      MemoryStatus = "active"
	MemoryStatusPendingSync MemoryStatus = "pending_sync"
	MemoryStatusSynced      MemoryStatus = "synced"
	MemoryStatusForgotten   MemoryStatus = "forgotten"
)

func (s MemoryStatus) Valid() bool {
	switch s {
	case MemoryStatusActive, MemoryStatusPendingSync, MemoryStatusSynced, MemoryStatusForgotten:
		return true
	default:
		return false
	}
}

type DedupAction string

const (
	DedupActionCreateNew DedupAction = "create_new"
	DedupActionReinforce DedupAction = "reinforce"
)

func (a DedupAction) Valid() bool {
	switch a {
	case DedupActionCreateNew, DedupActionReinforce:
		return true
	default:
		return false
	}
}

type RelationType string

const (
	RelationTypeReferences RelationType = "references"
	RelationTypeDuplicate  RelationType = "duplicate"
	RelationTypeReinforce  RelationType = "reinforce"
	RelationTypeComplement RelationType = "complement"
	RelationTypeUpdate     RelationType = "update"
	RelationTypeConflict   RelationType = "conflict"
	RelationTypeAssociated RelationType = "associated"
)

func (t RelationType) Valid() bool {
	switch t {
	case RelationTypeReferences, RelationTypeDuplicate, RelationTypeReinforce, RelationTypeComplement, RelationTypeUpdate, RelationTypeConflict, RelationTypeAssociated:
		return true
	default:
		return false
	}
}

type SourceRef struct {
	ID        string
	Kind      string
	URI       string
	Title     string
	CreatedAt time.Time
}

func (s SourceRef) Validate() error {
	if strings.TrimSpace(s.ID) == "" && strings.TrimSpace(s.URI) == "" && strings.TrimSpace(s.Title) == "" {
		return ErrEmptySourceRef
	}

	return nil
}

type Entity struct {
	ID             string
	Name           string
	Type           string
	Aliases        []string
	Confidence     float64
	SourceMemoryID string
	CreatedAt      time.Time
}

func (e Entity) Validate() error {
	if strings.TrimSpace(e.Name) == "" {
		return ErrEmptyName
	}

	if !isUnitInterval(e.Confidence) {
		return ErrInvalidConfidence
	}

	return nil
}

type Fact struct {
	ID             string
	Subject        string
	Predicate      string
	Object         string
	Confidence     float64
	SourceMemoryID string
	CreatedAt      time.Time
}

func (f Fact) Validate() error {
	if strings.TrimSpace(f.Subject) == "" || strings.TrimSpace(f.Predicate) == "" || strings.TrimSpace(f.Object) == "" {
		return ErrEmptyFactField
	}

	if !isUnitInterval(f.Confidence) {
		return ErrInvalidConfidence
	}

	return nil
}

type Event struct {
	ID             string
	Title          string
	Description    string
	OccurredAt     time.Time
	SourceMemoryID string
	CreatedAt      time.Time
}

func (e Event) Validate() error {
	if strings.TrimSpace(e.Title) == "" {
		return ErrEmptyName
	}

	return nil
}

type Concept struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
}

func (c Concept) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return ErrEmptyName
	}

	return nil
}

type Relation struct {
	ID         string
	FromID     string
	ToID       string
	Type       RelationType
	Confidence float64
	CreatedAt  time.Time
}

func (r Relation) Validate() error {
	if strings.TrimSpace(r.FromID) == "" || strings.TrimSpace(r.ToID) == "" {
		return ErrEmptyRelationSide
	}

	if !r.Type.Valid() {
		return ErrInvalidRelationType
	}

	if !isUnitInterval(r.Confidence) {
		return ErrInvalidConfidence
	}

	return nil
}

type Memory struct {
	ID                string
	Content           string
	NormalizedContent string
	Hash              string
	Kind              MemoryKind
	Status            MemoryStatus
	Scope             string
	Importance        float64
	Confidence        float64
	EmbeddingModel    string
	EmbeddingDim      int
	SourceRefs        []SourceRef
	Metadata          map[string]string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (m *Memory) Normalize() {
	if m.Kind == "" {
		m.Kind = MemoryKindNote
	}

	if m.Status == "" {
		m.Status = MemoryStatusPendingSync
	}

	if strings.TrimSpace(m.Scope) == "" {
		m.Scope = DefaultScope
	}

	if m.Metadata == nil {
		m.Metadata = map[string]string{}
	}
}

func (m Memory) Validate() error {
	if strings.TrimSpace(m.Content) == "" {
		return ErrEmptyContent
	}

	if !m.Kind.Valid() {
		return ErrInvalidMemoryKind
	}

	if !m.Status.Valid() {
		return ErrInvalidMemoryStatus
	}

	if !isUnitInterval(m.Importance) {
		return ErrInvalidImportance
	}

	if !isUnitInterval(m.Confidence) {
		return ErrInvalidConfidence
	}

	if m.EmbeddingDim < 0 {
		return errors.New("embedding dimension cannot be negative")
	}

	for _, ref := range m.SourceRefs {
		if err := ref.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func isUnitInterval(value float64) bool {
	return value >= 0 && value <= 1
}
