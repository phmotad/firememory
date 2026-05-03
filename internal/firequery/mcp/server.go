package mcp

import (
	"context"
	"errors"
	"sync"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

var ErrToolNotFound = errors.New("firequery/mcp: tool not found")

type ToolHandler func(ctx context.Context, request contract.ExternalRequest) (contract.ExternalResponse, error)

type Server struct {
	mu       sync.RWMutex
	handlers map[string]ToolHandler
}

func NewServer() *Server {
	return &Server{
		handlers: make(map[string]ToolHandler),
	}
}

func (s *Server) Register(name string, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[name] = handler
}

func (s *Server) RegisterDefaultTools(handler ToolHandler) {
	s.Register(ToolAsk, wrapAsk(handler))
	s.Register(ToolPlan, wrapPlan(handler))
	s.Register(ToolRemember, wrapOperation(handler, "remember"))
	s.Register(ToolRecall, wrapOperation(handler, "recall"))
	s.Register(ToolGetContext, wrapOperation(handler, "get_context"))
	s.Register(ToolExplain, wrapOperation(handler, "explain"))
}

func (s *Server) Handle(ctx context.Context, name string, request contract.ExternalRequest) (contract.ExternalResponse, error) {
	s.mu.RLock()
	handler, ok := s.handlers[name]
	s.mu.RUnlock()
	if !ok {
		return contract.ExternalResponse{}, ErrToolNotFound
	}

	return handler(ctx, request)
}

func wrapOperation(handler ToolHandler, operation string) ToolHandler {
	return func(ctx context.Context, request contract.ExternalRequest) (contract.ExternalResponse, error) {
		cloned := cloneRequest(request)
		cloned.Operation = operation
		return handler(ctx, cloned)
	}
}

func wrapAsk(handler ToolHandler) ToolHandler {
	return func(ctx context.Context, request contract.ExternalRequest) (contract.ExternalResponse, error) {
		cloned := cloneRequest(request)
		if cloned.Operation == "" {
			cloned.Operation = inferOperation(cloned)
		}
		return handler(ctx, cloned)
	}
}

func wrapPlan(handler ToolHandler) ToolHandler {
	return func(ctx context.Context, request contract.ExternalRequest) (contract.ExternalResponse, error) {
		cloned := cloneRequest(request)
		cloned.Operation = "get_context"
		if cloned.Input == nil {
			cloned.Input = map[string]any{}
		}
		cloned.Input["planning_mode"] = true
		cloned.Input["allow_write"] = false
		return handler(ctx, cloned)
	}
}

func inferOperation(request contract.ExternalRequest) string {
	if request.Input == nil {
		return "recall"
	}
	if content, ok := request.Input["content"].(string); ok && content != "" {
		return "remember"
	}
	if target, ok := request.Input["target_operation"].(string); ok && target != "" {
		return "explain"
	}
	if task, ok := request.Input["task"].(string); ok && task != "" {
		return "get_context"
	}
	return "recall"
}

func cloneRequest(request contract.ExternalRequest) contract.ExternalRequest {
	cloned := request
	if request.Input != nil {
		cloned.Input = make(map[string]any, len(request.Input))
		for key, value := range request.Input {
			cloned.Input[key] = value
		}
	} else {
		cloned.Input = map[string]any{}
	}
	return cloned
}
