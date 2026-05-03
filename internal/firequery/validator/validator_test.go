package validator

import (
	"context"
	"testing"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

func TestStrictValidatorValidateExternal(t *testing.T) {
	t.Parallel()

	validator := StrictValidator{}
	valid := contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_ext_1",
		Language:  "en",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Operation: "get_context",
		Brain:     "agent.fbrain",
		Input:     map[string]any{"task": "responder cliente"},
	}

	if result := validator.ValidateExternal(valid); !result.OK {
		t.Fatalf("ValidateExternal(valid) = %#v", result)
	}

	tests := []struct {
		name    string
		request contract.ExternalRequest
		want    string
	}{
		{
			name:    "missing version",
			request: contract.ExternalRequest{},
			want:    "Missing required field: version",
		},
		{
			name: "language must be english",
			request: contract.ExternalRequest{
				Version:   "0.1",
				RequestID: "req_ext_lang",
				Language:  "pt-BR",
				Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
				Operation: "recall",
				Brain:     "agent.fbrain",
				Input:     map[string]any{},
			},
			want: "Invalid language: expected en",
		},
		{
			name: "invalid brain",
			request: contract.ExternalRequest{
				Version:   "0.1",
				RequestID: "req_ext_2",
				Language:  "en",
				Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
				Operation: "recall",
				Brain:     "agent.brain",
				Input:     map[string]any{},
			},
			want: "Invalid brain extension: must end with .fbrain",
		},
		{
			name: "missing input",
			request: contract.ExternalRequest{
				Version:   "0.1",
				RequestID: "req_ext_3",
				Language:  "en",
				Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
				Operation: "recall",
				Brain:     "agent.fbrain",
			},
			want: "Missing required field: input",
		},
		{
			name: "unknown operation",
			request: contract.ExternalRequest{
				Version:   "0.1",
				RequestID: "req_ext_4",
				Language:  "en",
				Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
				Operation: "sql_query",
				Brain:     "agent.fbrain",
				Input:     map[string]any{},
			},
			want: "Unknown operation: sql_query",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := validator.ValidateExternal(tt.request)
			if result.OK || !result.Rejected {
				t.Fatalf("ValidateExternal() = %#v, want rejected", result)
			}
			if result.Error == nil || result.Error.Message != tt.want {
				t.Fatalf("error message = %#v, want %q", result.Error, tt.want)
			}
		})
	}
}

func TestStrictValidatorValidateInternal(t *testing.T) {
	t.Parallel()

	validator := StrictValidator{}
	valid := contract.OperationRequest{
		Version:   "0.1",
		RequestID: "req_int_1",
		Language:  "en",
		Actor:     contract.Actor{Type: "firequery", ID: "firequery-mcp"},
		Operation: "get_context",
		Intent:    "build_context",
		Brain:     "agent.fbrain",
		Scope:     "default",
		Input:     map[string]any{"task": "answer Joao"},
		Permissions: &contract.Permissions{
			AllowWrite:           false,
			RequiresConfirmation: false,
		},
		Thresholds: &contract.Thresholds{
			TopK:                8,
			SimilarityThreshold: 0.7,
			BudgetTokens:        1500,
		},
	}

	if result := validator.ValidateInternal(valid); !result.OK {
		t.Fatalf("ValidateInternal(valid) = %#v", result)
	}

	tests := []struct {
		name    string
		request contract.OperationRequest
		want    string
	}{
		{
			name: "language must be english",
			request: contract.OperationRequest{
				Version:     "0.1",
				RequestID:   "req_2",
				Language:    "pt-BR",
				Actor:       contract.Actor{Type: "firequery", ID: "firequery-mcp"},
				Operation:   "recall",
				Intent:      "recall_information",
				Brain:       "agent.fbrain",
				Scope:       "default",
				Permissions: &contract.Permissions{},
			},
			want: "Invalid language: expected en",
		},
		{
			name: "missing permissions",
			request: contract.OperationRequest{
				Version:   "0.1",
				RequestID: "req_3",
				Language:  "en",
				Actor:     contract.Actor{Type: "firequery", ID: "firequery-mcp"},
				Operation: "recall",
				Intent:    "recall_information",
				Brain:     "agent.fbrain",
				Scope:     "default",
			},
			want: "Missing required field: permissions",
		},
		{
			name: "intent mismatch",
			request: contract.OperationRequest{
				Version:   "0.1",
				RequestID: "req_4",
				Language:  "en",
				Actor:     contract.Actor{Type: "firequery", ID: "firequery-mcp"},
				Operation: "recall",
				Intent:    "build_context",
				Brain:     "agent.fbrain",
				Scope:     "default",
				Permissions: &contract.Permissions{
					AllowWrite: false,
				},
			},
			want: "Intent does not match operation: expected recall_information",
		},
		{
			name: "invalid threshold range",
			request: contract.OperationRequest{
				Version:   "0.1",
				RequestID: "req_5",
				Language:  "en",
				Actor:     contract.Actor{Type: "firequery", ID: "firequery-mcp"},
				Operation: "recall",
				Intent:    "recall_information",
				Brain:     "agent.fbrain",
				Scope:     "default",
				Permissions: &contract.Permissions{
					AllowWrite: false,
				},
				Thresholds: &contract.Thresholds{
					TopK:                8,
					SimilarityThreshold: 1.5,
					BudgetTokens:        1000,
				},
			},
			want: "Invalid thresholds: similarity_threshold must be between 0 and 1",
		},
		{
			name: "write denied",
			request: contract.OperationRequest{
				Version:   "0.1",
				RequestID: "req_6",
				Language:  "en",
				Actor:     contract.Actor{Type: "firequery", ID: "firequery-mcp"},
				Operation: "remember",
				Intent:    "remember_information",
				Brain:     "agent.fbrain",
				Scope:     "default",
				Permissions: &contract.Permissions{
					AllowWrite: false,
				},
			},
			want: "Write operation requires allow_write = true",
		},
		{
			name: "forget needs confirmation",
			request: contract.OperationRequest{
				Version:   "0.1",
				RequestID: "req_7",
				Language:  "en",
				Actor:     contract.Actor{Type: "firequery", ID: "firequery-mcp"},
				Operation: "forget",
				Intent:    "forget_memory",
				Brain:     "agent.fbrain",
				Scope:     "default",
				Permissions: &contract.Permissions{
					AllowWrite:           true,
					RequiresConfirmation: false,
				},
			},
			want: "Forget requires confirmation",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := validator.ValidateInternal(tt.request)
			if result.OK || !result.Rejected {
				t.Fatalf("ValidateInternal() = %#v, want rejected", result)
			}
			if result.Error == nil || result.Error.Message != tt.want {
				t.Fatalf("error message = %#v, want %q", result.Error, tt.want)
			}
		})
	}
}

func TestGuardedClientRejectsBeforeCall(t *testing.T) {
	t.Parallel()

	client := &recordingClient{}
	guard := GuardedClient{
		Validator: StrictValidator{},
		Client:    client,
	}

	response, err := guard.Call(context.Background(), contract.OperationRequest{
		Version:   "0.1",
		RequestID: "req_invalid",
		Language:  "pt-BR",
		Actor:     contract.Actor{Type: "firequery", ID: "firequery-mcp"},
		Operation: "recall",
		Intent:    "recall_information",
		Brain:     "agent.fbrain",
		Scope:     "default",
		Permissions: &contract.Permissions{
			AllowWrite: false,
		},
	})
	if err != nil {
		t.Fatalf("GuardedClient.Call() error = %v", err)
	}
	if client.called {
		t.Fatal("client should not have been called")
	}
	if response.OK || !response.Rejected {
		t.Fatalf("response = %#v, want rejected", response)
	}
	if response.Error == nil || response.Error.Code != codeContractValidationFailed {
		t.Fatalf("response error = %#v", response.Error)
	}
}

func TestGuardedClientCallsFireMemoryForValidRequest(t *testing.T) {
	t.Parallel()

	client := &recordingClient{
		response: contract.OperationResponse{
			OK:        true,
			RequestID: "req_valid",
			Operation: "recall",
		},
	}
	guard := GuardedClient{
		Validator: StrictValidator{},
		Client:    client,
	}

	response, err := guard.Call(context.Background(), contract.OperationRequest{
		Version:   "0.1",
		RequestID: "req_valid",
		Language:  "en",
		Actor:     contract.Actor{Type: "firequery", ID: "firequery-mcp"},
		Operation: "recall",
		Intent:    "recall_information",
		Brain:     "agent.fbrain",
		Scope:     "default",
		Permissions: &contract.Permissions{
			AllowWrite: false,
		},
	})
	if err != nil {
		t.Fatalf("GuardedClient.Call() error = %v", err)
	}
	if !client.called {
		t.Fatal("client should have been called")
	}
	if !response.OK {
		t.Fatalf("response = %#v, want ok", response)
	}
}

type recordingClient struct {
	called   bool
	response contract.OperationResponse
}

func (c *recordingClient) Call(context.Context, contract.OperationRequest) (contract.OperationResponse, error) {
	c.called = true
	return c.response, nil
}
