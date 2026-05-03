package engine

import (
	"errors"
	"strings"

	"github.com/phmotad/firememory/internal/memory"
)

const (
	DefaultTopK         = 8
	DefaultBudgetTokens = 2000
)

var (
	ErrBrainPathRequired    = errors.New("brain path is required")
	ErrBrainPathExtension   = errors.New("brain path must end with .fbrain")
	ErrQueryRequired        = errors.New("query is required")
	ErrContentRequired      = errors.New("content is required")
	ErrMemoryIDRequired     = errors.New("memory id is required")
	ErrOperationRequired    = errors.New("operation is required")
	ErrInvalidTopK          = errors.New("top_k must be greater than zero")
	ErrInvalidBudgetTokens  = errors.New("budget_tokens must be greater than zero")
	ErrInvalidLimit         = errors.New("limit cannot be negative")
	ErrConfirmationRequired = errors.New("confirmation is required")
)

type RememberInput struct {
	BrainPath  string             `json:"brain_path"`
	Content    string             `json:"content"`
	Scope      string             `json:"scope"`
	Kind       memory.MemoryKind  `json:"kind"`
	SourceRefs []memory.SourceRef `json:"source_refs"`
	Metadata   map[string]string  `json:"metadata"`
}

func (in *RememberInput) Normalize() {
	if in.Kind == "" {
		in.Kind = memory.MemoryKindNote
	}

	if strings.TrimSpace(in.Scope) == "" {
		in.Scope = memory.DefaultScope
	}
}

func (in RememberInput) Validate() error {
	if err := validateBrainPath(in.BrainPath); err != nil {
		return err
	}

	if strings.TrimSpace(in.Content) == "" {
		return ErrContentRequired
	}

	if in.Kind != "" && !in.Kind.Valid() {
		return memory.ErrInvalidMemoryKind
	}

	for _, ref := range in.SourceRefs {
		if err := ref.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type RememberResult struct {
	Memory             memory.Memory
	DedupAction        memory.DedupAction
	ReinforcedMemoryID string
	Trace              []string
}

type RecallInput struct {
	BrainPath    string `json:"brain_path"`
	Query        string `json:"query"`
	Scope        string `json:"scope"`
	TopK         int    `json:"top_k"`
	IncludeTrace bool   `json:"include_trace"`
}

func (in *RecallInput) Normalize() {
	if strings.TrimSpace(in.Scope) == "" {
		in.Scope = memory.DefaultScope
	}

	if in.TopK == 0 {
		in.TopK = DefaultTopK
	}
}

func (in RecallInput) Validate() error {
	if err := validateBrainPath(in.BrainPath); err != nil {
		return err
	}

	if strings.TrimSpace(in.Query) == "" {
		return ErrQueryRequired
	}

	if in.TopK < 0 {
		return ErrInvalidTopK
	}

	return nil
}

type RecallHit struct {
	Memory  memory.Memory
	Score   float64
	Reasons []string
}

type RecallResult struct {
	Hits  []RecallHit
	Trace []string
}

type RelateInput struct {
	BrainPath string              `json:"brain_path"`
	FromID    string              `json:"from_id"`
	ToID      string              `json:"to_id"`
	Type      memory.RelationType `json:"type"`
}

func (in RelateInput) Validate() error {
	if err := validateBrainPath(in.BrainPath); err != nil {
		return err
	}

	if strings.TrimSpace(in.FromID) == "" || strings.TrimSpace(in.ToID) == "" {
		return ErrMemoryIDRequired
	}

	if !in.Type.Valid() {
		return memory.ErrInvalidRelationType
	}

	return nil
}

type RelateResult struct {
	Relation memory.Relation
	Trace    []string
}

type ForgetInput struct {
	BrainPath            string `json:"brain_path"`
	MemoryID             string `json:"memory_id"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

func (in ForgetInput) Validate() error {
	if err := validateBrainPath(in.BrainPath); err != nil {
		return err
	}

	if strings.TrimSpace(in.MemoryID) == "" {
		return ErrMemoryIDRequired
	}

	if !in.RequiresConfirmation {
		return ErrConfirmationRequired
	}

	return nil
}

type ForgetResult struct {
	Forgotten bool
	Trace     []string
}

type ConsolidateInput struct {
	BrainPath string `json:"brain_path"`
	Scope     string `json:"scope"`
	Limit     int    `json:"limit"`
}

func (in *ConsolidateInput) Normalize() {
	if strings.TrimSpace(in.Scope) == "" {
		in.Scope = memory.DefaultScope
	}
}

func (in ConsolidateInput) Validate() error {
	if err := validateBrainPath(in.BrainPath); err != nil {
		return err
	}

	if in.Limit < 0 {
		return ErrInvalidLimit
	}

	return nil
}

type ConsolidateResult struct {
	Processed int
	Trace     []string
}

type SyncInput struct {
	BrainPath string `json:"brain_path"`
	Limit     int    `json:"limit"`
}

func (in SyncInput) Validate() error {
	if err := validateBrainPath(in.BrainPath); err != nil {
		return err
	}

	if in.Limit < 0 {
		return ErrInvalidLimit
	}

	return nil
}

type SyncResult struct {
	Processed int
	SyncedIDs []string
	Trace     []string
}

type ContextInput struct {
	BrainPath    string `json:"brain_path"`
	Query        string `json:"query"`
	Scope        string `json:"scope"`
	TopK         int    `json:"top_k"`
	BudgetTokens int    `json:"budget_tokens"`
	IncludeGraph bool   `json:"include_graph"`
	IncludeTrace bool   `json:"include_trace"`
}

func (in *ContextInput) Normalize() {
	if strings.TrimSpace(in.Scope) == "" {
		in.Scope = memory.DefaultScope
	}

	if in.TopK == 0 {
		in.TopK = DefaultTopK
	}

	if in.BudgetTokens == 0 {
		in.BudgetTokens = DefaultBudgetTokens
	}
}

func (in ContextInput) Validate() error {
	if err := validateBrainPath(in.BrainPath); err != nil {
		return err
	}

	if strings.TrimSpace(in.Query) == "" {
		return ErrQueryRequired
	}

	if in.TopK < 0 {
		return ErrInvalidTopK
	}

	if in.BudgetTokens < 0 {
		return ErrInvalidBudgetTokens
	}

	return nil
}

type ContextResult struct {
	Memories        []memory.Memory
	Entities        []memory.Entity
	Facts           []memory.Fact
	Relations       []memory.Relation
	ContextText     string
	EstimatedTokens int
	Trace           []string
}

type ExplainInput struct {
	BrainPath string   `json:"brain_path"`
	Operation string   `json:"operation"`
	MemoryID  string   `json:"memory_id"`
	Trace     []string `json:"trace"`
}

func (in ExplainInput) Validate() error {
	if err := validateBrainPath(in.BrainPath); err != nil {
		return err
	}

	if strings.TrimSpace(in.Operation) == "" {
		return ErrOperationRequired
	}

	return nil
}

type ExplainResult struct {
	Operation string
	Summary   string
	Trace     []string
}

func validateBrainPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return ErrBrainPathRequired
	}

	if !strings.HasSuffix(strings.ToLower(path), ".fbrain") {
		return ErrBrainPathExtension
	}

	return nil
}
