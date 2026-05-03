package adapters

import (
	"context"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

type FireMemoryClient interface {
	Call(ctx context.Context, request contract.OperationRequest) (contract.OperationResponse, error)
}

type NoopFireMemoryClient struct{}

func (NoopFireMemoryClient) Call(context.Context, contract.OperationRequest) (contract.OperationResponse, error) {
	return contract.OperationResponse{OK: true}, nil
}
