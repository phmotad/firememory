package builder

import (
	"testing"

	"github.com/phmotad/firememory/internal/firequery/contract"
	"github.com/phmotad/firememory/internal/firequery/models"
)

func TestGoContractBuilderBuildsInternalRequest(t *testing.T) {
	t.Parallel()

	b := NewGoContractBuilder("")
	request := contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_1",
		Language:  "pt-BR",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Operation: "get_context",
		Brain:     "agent.fbrain",
		Input: map[string]any{
			"task":                 "O Joao ainda usa Firebird 2.5?",
			"budget_tokens":        1200,
			"include_graph":        true,
			"include_trace":        true,
			"scope":                "default",
			"allow_write":          false,
			"top_k":                6,
			"similarity_threshold": 0.8,
		},
	}

	internal := b.Build(request, Inputs{
		Intent:  models.IntentResult{Intent: "build_context", Score: 0.95},
		Trigger: models.TriggerResult{Trigger: "query_memory", Score: 0.9},
	})

	if internal.Language != "en" {
		t.Fatalf("language = %q, want en", internal.Language)
	}
	if internal.Intent != "build_context" {
		t.Fatalf("intent = %q", internal.Intent)
	}
	if internal.Input["query"] != "O Joao ainda usa Firebird 2.5?" {
		t.Fatalf("query = %#v", internal.Input["query"])
	}
	if internal.Permissions == nil || internal.Permissions.AllowWrite {
		t.Fatalf("permissions = %#v, want allow_write=false", internal.Permissions)
	}
	if internal.Thresholds == nil || internal.Thresholds.TopK != 6 {
		t.Fatalf("thresholds = %#v", internal.Thresholds)
	}
}

func TestGoContractBuilderRequiresExplicitWriteOptIn(t *testing.T) {
	t.Parallel()

	b := NewGoContractBuilder("firequery-mcp")
	request := contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_2",
		Language:  "en",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Operation: "remember",
		Brain:     "agent.fbrain",
		Input: map[string]any{
			"content": "Remember this",
		},
	}

	internal := b.Build(request, Inputs{
		Intent:  models.IntentResult{Intent: "remember_information", Score: 1},
		Trigger: models.TriggerResult{Trigger: "suggest_write", Score: 1},
	})

	if internal.Permissions == nil || internal.Permissions.AllowWrite {
		t.Fatalf("permissions = %#v, want explicit allow_write=false", internal.Permissions)
	}
}
