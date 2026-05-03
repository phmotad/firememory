package memory

import "testing"

func TestMemoryNormalizeAppliesDefaults(t *testing.T) {
	m := &Memory{
		Content:    "Cliente usa Firebird 2.5",
		Importance: 0.6,
		Confidence: 0.8,
	}

	m.Normalize()

	if m.Kind != MemoryKindNote {
		t.Fatalf("expected default kind %q, got %q", MemoryKindNote, m.Kind)
	}

	if m.Status != MemoryStatusPendingSync {
		t.Fatalf("expected default status %q, got %q", MemoryStatusPendingSync, m.Status)
	}

	if m.Scope != DefaultScope {
		t.Fatalf("expected default scope %q, got %q", DefaultScope, m.Scope)
	}

	if m.Metadata == nil {
		t.Fatal("expected metadata map to be initialized")
	}
}

func TestMemoryValidateRejectsInvalidFields(t *testing.T) {
	m := Memory{
		Content:    "x",
		Kind:       MemoryKind("broken"),
		Status:     MemoryStatusPendingSync,
		Importance: 0.5,
		Confidence: 0.7,
	}

	if err := m.Validate(); err != ErrInvalidMemoryKind {
		t.Fatalf("expected ErrInvalidMemoryKind, got %v", err)
	}

	m.Kind = MemoryKindNote
	m.Status = MemoryStatus("broken")

	if err := m.Validate(); err != ErrInvalidMemoryStatus {
		t.Fatalf("expected ErrInvalidMemoryStatus, got %v", err)
	}
}

func TestMemoryValidateAcceptsValidMemory(t *testing.T) {
	m := Memory{
		ID:         "mem_01",
		Content:    "Cliente Joao teve erro fiscal na NF-e",
		Kind:       MemoryKindFact,
		Status:     MemoryStatusPendingSync,
		Scope:      DefaultScope,
		Importance: 0.4,
		Confidence: 0.9,
		SourceRefs: []SourceRef{
			{ID: "src_01"},
		},
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("expected valid memory, got error %v", err)
	}
}

func TestRelationValidate(t *testing.T) {
	r := Relation{
		FromID:     "mem_01",
		ToID:       "mem_02",
		Type:       RelationTypeComplement,
		Confidence: 0.75,
	}

	if err := r.Validate(); err != nil {
		t.Fatalf("expected valid relation, got error %v", err)
	}
}

func TestFactValidateRequiresTriple(t *testing.T) {
	f := Fact{
		Subject:    "Joao",
		Predicate:  "",
		Object:     "Firebird 2.5",
		Confidence: 0.9,
	}

	if err := f.Validate(); err != ErrEmptyFactField {
		t.Fatalf("expected ErrEmptyFactField, got %v", err)
	}
}

func TestEnumValidation(t *testing.T) {
	if !DedupActionCreateNew.Valid() || !DedupActionReinforce.Valid() {
		t.Fatal("expected dedup actions to be valid")
	}

	if RelationType("unknown").Valid() {
		t.Fatal("expected unknown relation type to be invalid")
	}
}
