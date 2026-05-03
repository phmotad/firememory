package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

const (
	codeExternalContractValidationFailed = "EXTERNAL_CONTRACT_VALIDATION_FAILED"
	codeContractValidationFailed         = "CONTRACT_VALIDATION_FAILED"
)

var operationToIntent = map[string]string{
	"remember":    "remember_information",
	"recall":      "recall_information",
	"relate":      "relate_memory",
	"forget":      "forget_memory",
	"consolidate": "consolidate_memory",
	"get_context": "build_context",
	"explain":     "explain_decision",
	"sync":        "sync_memory",
}

type Result struct {
	OK       bool
	Rejected bool
	Error    *contract.Error
}

type ExternalValidator interface {
	ValidateExternal(request contract.ExternalRequest) Result
}

type InternalValidator interface {
	ValidateInternal(request contract.OperationRequest) Result
}

type Validator interface {
	ExternalValidator
	InternalValidator
}

type FireMemoryCaller interface {
	Call(ctx context.Context, request contract.OperationRequest) (contract.OperationResponse, error)
}

type NoopValidator struct{}

func (NoopValidator) ValidateExternal(contract.ExternalRequest) Result {
	return Result{OK: true}
}

func (NoopValidator) ValidateInternal(contract.OperationRequest) Result {
	return Result{OK: true}
}

type StrictValidator struct{}

func (StrictValidator) ValidateExternal(request contract.ExternalRequest) Result {
	if strings.TrimSpace(request.Version) == "" {
		return reject(codeExternalContractValidationFailed, "Missing required field: version")
	}
	if strings.TrimSpace(request.RequestID) == "" {
		return reject(codeExternalContractValidationFailed, "Missing required field: request_id")
	}
	if request.Language != "en" {
		return reject(codeExternalContractValidationFailed, "Invalid language: expected en")
	}
	if strings.TrimSpace(request.Actor.Type) == "" || strings.TrimSpace(request.Actor.ID) == "" {
		return reject(codeExternalContractValidationFailed, "Missing required field: actor")
	}
	if strings.TrimSpace(request.Operation) == "" {
		return reject(codeExternalContractValidationFailed, "Missing required field: operation")
	}
	if !isKnownOperation(request.Operation) {
		return reject(codeExternalContractValidationFailed, fmt.Sprintf("Unknown operation: %s", request.Operation))
	}
	if strings.TrimSpace(request.Brain) == "" {
		return reject(codeExternalContractValidationFailed, "Missing required field: brain")
	}
	if !strings.HasSuffix(request.Brain, ".fbrain") {
		return reject(codeExternalContractValidationFailed, "Invalid brain extension: must end with .fbrain")
	}
	if request.Input == nil {
		return reject(codeExternalContractValidationFailed, "Missing required field: input")
	}

	return Result{OK: true}
}

func (StrictValidator) ValidateInternal(request contract.OperationRequest) Result {
	if strings.TrimSpace(request.Version) == "" {
		return reject(codeContractValidationFailed, "Missing required field: version")
	}
	if strings.TrimSpace(request.RequestID) == "" {
		return reject(codeContractValidationFailed, "Missing required field: request_id")
	}
	if request.Language != "en" {
		return reject(codeContractValidationFailed, "Invalid language: expected en")
	}
	if strings.TrimSpace(request.Actor.Type) == "" || strings.TrimSpace(request.Actor.ID) == "" {
		return reject(codeContractValidationFailed, "Missing required field: actor")
	}
	if strings.TrimSpace(request.Operation) == "" {
		return reject(codeContractValidationFailed, "Missing required field: operation")
	}
	if !isKnownOperation(request.Operation) {
		return reject(codeContractValidationFailed, fmt.Sprintf("Unknown operation: %s", request.Operation))
	}
	if strings.TrimSpace(request.Intent) == "" {
		return reject(codeContractValidationFailed, "Missing required field: intent")
	}
	if wantIntent := operationToIntent[request.Operation]; request.Intent != wantIntent {
		return reject(codeContractValidationFailed, fmt.Sprintf("Intent does not match operation: expected %s", wantIntent))
	}
	if strings.TrimSpace(request.Brain) == "" {
		return reject(codeContractValidationFailed, "Missing required field: brain")
	}
	if !strings.HasSuffix(request.Brain, ".fbrain") {
		return reject(codeContractValidationFailed, "Invalid brain extension: must end with .fbrain")
	}
	if strings.TrimSpace(request.Scope) == "" {
		return reject(codeContractValidationFailed, "Missing required field: scope")
	}
	if request.Permissions == nil {
		return reject(codeContractValidationFailed, "Missing required field: permissions")
	}
	if err := validateThresholds(request.Thresholds); err != nil {
		return reject(codeContractValidationFailed, err.Error())
	}
	if err := validatePermissions(request); err != nil {
		return reject(codeContractValidationFailed, err.Error())
	}

	return Result{OK: true}
}

type GuardedClient struct {
	Validator InternalValidator
	Client    FireMemoryCaller
}

func (c GuardedClient) Call(ctx context.Context, request contract.OperationRequest) (contract.OperationResponse, error) {
	if result := c.Validator.ValidateInternal(request); !result.OK {
		return contract.OperationResponse{
			OK:        false,
			RequestID: request.RequestID,
			Operation: request.Operation,
			Rejected:  true,
			Error:     result.Error,
		}, nil
	}

	return c.Client.Call(ctx, request)
}

func IsWriteOperation(operation string) bool {
	switch operation {
	case "remember", "relate", "forget", "consolidate", "sync":
		return true
	default:
		return false
	}
}

func isKnownOperation(operation string) bool {
	_, ok := operationToIntent[operation]
	return ok
}

func validateThresholds(thresholds *contract.Thresholds) error {
	if thresholds == nil {
		return nil
	}
	if thresholds.TopK <= 0 {
		return fmt.Errorf("Invalid thresholds: top_k must be greater than zero")
	}
	if thresholds.SimilarityThreshold < 0 || thresholds.SimilarityThreshold > 1 {
		return fmt.Errorf("Invalid thresholds: similarity_threshold must be between 0 and 1")
	}
	if thresholds.BudgetTokens <= 0 {
		return fmt.Errorf("Invalid thresholds: budget_tokens must be greater than zero")
	}
	return nil
}

func validatePermissions(request contract.OperationRequest) error {
	if request.Permissions == nil {
		return fmt.Errorf("Missing required field: permissions")
	}
	if IsWriteOperation(request.Operation) && !request.Permissions.AllowWrite {
		return fmt.Errorf("Write operation requires allow_write = true")
	}
	if request.Operation == "forget" && !request.Permissions.RequiresConfirmation {
		return fmt.Errorf("Forget requires confirmation")
	}
	return nil
}

func reject(code, message string) Result {
	return Result{
		OK:       false,
		Rejected: true,
		Error: &contract.Error{
			Code:    code,
			Message: message,
		},
	}
}
