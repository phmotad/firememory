package adapters

import (
	"context"
	"fmt"

	"github.com/phmotad/firememory/internal/engine"
	"github.com/phmotad/firememory/internal/firequery/contract"
)

// PersistentEngineClient implements FireMemoryClient using a pre-opened engine
// that remains open for the lifetime of the daemon. Unlike EngineClient it
// never calls engine.Open or engine.Close — ownership stays with the caller.
type PersistentEngineClient struct {
	eng engine.Engine
}

func NewPersistentEngineClient(eng engine.Engine) PersistentEngineClient {
	return PersistentEngineClient{eng: eng}
}

func (c PersistentEngineClient) Call(_ context.Context, request contract.OperationRequest) (contract.OperationResponse, error) {
	switch request.Operation {
	case "remember":
		return callRemember(c.eng, request)
	case "recall":
		return callRecall(c.eng, request)
	case "get_context":
		return callContext(c.eng, request)
	case "explain":
		return callExplain(c.eng, request)
	case "sync":
		return callSync(c.eng, request)
	default:
		return contract.OperationResponse{}, fmt.Errorf("firequery/adapters: unsupported operation %q", request.Operation)
	}
}
