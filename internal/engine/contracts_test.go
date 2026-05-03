package engine

import (
	"testing"

	"github.com/phmotad/firememory/internal/memory"
)

func TestRememberInputNormalizeAndValidate(t *testing.T) {
	in := &RememberInput{
		BrainPath: "agent.fbrain",
		Content:   "Cliente Joao usa Firebird 2.5",
	}

	in.Normalize()

	if in.Kind != memory.MemoryKindNote {
		t.Fatalf("expected default kind %q, got %q", memory.MemoryKindNote, in.Kind)
	}

	if in.Scope != memory.DefaultScope {
		t.Fatalf("expected default scope %q, got %q", memory.DefaultScope, in.Scope)
	}

	if err := in.Validate(); err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
}

func TestRecallInputDefaults(t *testing.T) {
	in := &RecallInput{
		BrainPath: "agent.fbrain",
		Query:     "erro fiscal NF-e",
	}

	in.Normalize()

	if in.TopK != DefaultTopK {
		t.Fatalf("expected default top_k %d, got %d", DefaultTopK, in.TopK)
	}

	if err := in.Validate(); err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
}

func TestForgetInputRequiresConfirmation(t *testing.T) {
	in := ForgetInput{
		BrainPath: "agent.fbrain",
		MemoryID:  "mem_01",
	}

	if err := in.Validate(); err != ErrConfirmationRequired {
		t.Fatalf("expected ErrConfirmationRequired, got %v", err)
	}
}

func TestContextInputDefaults(t *testing.T) {
	in := &ContextInput{
		BrainPath: "agent.fbrain",
		Query:     "responder Joao sobre erro fiscal",
	}

	in.Normalize()

	if in.TopK != DefaultTopK {
		t.Fatalf("expected default top_k %d, got %d", DefaultTopK, in.TopK)
	}

	if in.BudgetTokens != DefaultBudgetTokens {
		t.Fatalf("expected default budget %d, got %d", DefaultBudgetTokens, in.BudgetTokens)
	}

	if err := in.Validate(); err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
}

func TestValidateBrainPathRequiresFbrain(t *testing.T) {
	err := validateBrainPath("agent.txt")
	if err != ErrBrainPathExtension {
		t.Fatalf("expected ErrBrainPathExtension, got %v", err)
	}
}
