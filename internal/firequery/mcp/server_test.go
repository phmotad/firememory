package mcp

import (
	"context"
	"testing"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

func TestServerHandle(t *testing.T) {
	t.Parallel()

	server := NewServer()
	server.Register("firequery.recall", func(_ context.Context, request contract.ExternalRequest) (contract.ExternalResponse, error) {
		return contract.ExternalResponse{
			OK:        true,
			RequestID: request.RequestID,
			Operation: request.Operation,
		}, nil
	})

	response, err := server.Handle(context.Background(), "firequery.recall", contract.ExternalRequest{
		RequestID: "req_1",
		Operation: "recall",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !response.OK {
		t.Fatal("expected ok response")
	}
	if response.RequestID != "req_1" {
		t.Fatalf("RequestID = %q, want req_1", response.RequestID)
	}
}

func TestServerHandleMissingTool(t *testing.T) {
	t.Parallel()

	_, err := NewServer().Handle(context.Background(), "missing", contract.ExternalRequest{})
	if err != ErrToolNotFound {
		t.Fatalf("Handle() error = %v, want %v", err, ErrToolNotFound)
	}
}

func TestRegisterDefaultTools(t *testing.T) {
	t.Parallel()

	server := NewServer()
	var seen []contract.ExternalRequest
	server.RegisterDefaultTools(func(_ context.Context, request contract.ExternalRequest) (contract.ExternalResponse, error) {
		seen = append(seen, request)
		return contract.ExternalResponse{
			OK:        true,
			RequestID: request.RequestID,
			Operation: request.Operation,
		}, nil
	})

	tests := []struct {
		name          string
		request       contract.ExternalRequest
		wantOperation string
	}{
		{
			name:          ToolRemember,
			request:       contract.ExternalRequest{RequestID: "req_1", Input: map[string]any{"content": "remember this"}},
			wantOperation: "remember",
		},
		{
			name:          ToolRecall,
			request:       contract.ExternalRequest{RequestID: "req_2", Input: map[string]any{"query": "find this"}},
			wantOperation: "recall",
		},
		{
			name:          ToolGetContext,
			request:       contract.ExternalRequest{RequestID: "req_3", Input: map[string]any{"task": "answer client"}},
			wantOperation: "get_context",
		},
		{
			name:          ToolExplain,
			request:       contract.ExternalRequest{RequestID: "req_4", Input: map[string]any{"target_operation": "recall"}},
			wantOperation: "explain",
		},
		{
			name:          ToolAsk,
			request:       contract.ExternalRequest{RequestID: "req_5", Input: map[string]any{"task": "answer client"}},
			wantOperation: "get_context",
		},
		{
			name:          ToolPlan,
			request:       contract.ExternalRequest{RequestID: "req_6", Input: map[string]any{"task": "prepare response"}},
			wantOperation: "get_context",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			response, err := server.Handle(context.Background(), tt.name, tt.request)
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if response.Operation != tt.wantOperation {
				t.Fatalf("Operation = %q, want %q", response.Operation, tt.wantOperation)
			}
		})
	}

	if len(seen) != len(tests) {
		t.Fatalf("seen requests = %d, want %d", len(seen), len(tests))
	}
	if planningMode, ok := seen[len(seen)-1].Input["planning_mode"].(bool); !ok || !planningMode {
		t.Fatalf("planning_mode = %#v, want true", seen[len(seen)-1].Input["planning_mode"])
	}
}

func TestDefaultSchemasContainAllTools(t *testing.T) {
	t.Parallel()

	schemas := DefaultSchemas()
	for _, tool := range []string{ToolAsk, ToolPlan, ToolRemember, ToolRecall, ToolGetContext, ToolExplain} {
		if _, ok := schemas[tool]; !ok {
			t.Fatalf("missing schema for %s", tool)
		}
	}
}
