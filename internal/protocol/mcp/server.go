package mcp

import (
	"encoding/json"
	"errors"

	"github.com/phmotad/firememory/internal/engine"
)

type ToolName string

const (
	ToolRemember   ToolName = "firememory.remember"
	ToolRecall     ToolName = "firememory.recall"
	ToolGetContext ToolName = "firememory.get_context"
	ToolSync       ToolName = "firememory.sync"
	ToolExplain    ToolName = "firememory.explain"
)

var (
	ErrUnknownTool = errors.New("unknown mcp tool")
)

type ToolDefinition struct {
	Name        ToolName
	Description string
	InputSchema map[string]any
}

type CallRequest struct {
	Tool      ToolName       `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

type CallResponse struct {
	Tool    ToolName `json:"tool"`
	OK      bool     `json:"ok"`
	Content any      `json:"content,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type Server struct {
	engine engine.Engine
	tools  map[ToolName]ToolDefinition
}

func NewServer(eng engine.Engine) *Server {
	return &Server{
		engine: eng,
		tools:  DefaultTools(),
	}
}

func (s *Server) Tools() []ToolDefinition {
	out := make([]ToolDefinition, 0, len(s.tools))
	for _, tool := range s.tools {
		out = append(out, tool)
	}
	return out
}

func (s *Server) HandleCall(req CallRequest) (CallResponse, error) {
	if _, ok := s.tools[req.Tool]; !ok {
		return CallResponse{}, ErrUnknownTool
	}

	switch req.Tool {
	case ToolRemember:
		var input engine.RememberInput
		if err := decodeArguments(req.Arguments, &input); err != nil {
			return errorResponse(req.Tool, err), nil
		}
		result, err := s.engine.Remember(input)
		if err != nil {
			return errorResponse(req.Tool, err), nil
		}
		return okResponse(req.Tool, result), nil

	case ToolRecall:
		var input engine.RecallInput
		if err := decodeArguments(req.Arguments, &input); err != nil {
			return errorResponse(req.Tool, err), nil
		}
		result, err := s.engine.Recall(input)
		if err != nil {
			return errorResponse(req.Tool, err), nil
		}
		return okResponse(req.Tool, result), nil

	case ToolGetContext:
		var input engine.ContextInput
		if err := decodeArguments(req.Arguments, &input); err != nil {
			return errorResponse(req.Tool, err), nil
		}
		result, err := s.engine.Context(input)
		if err != nil {
			return errorResponse(req.Tool, err), nil
		}
		return okResponse(req.Tool, result), nil

	case ToolSync:
		var input engine.SyncInput
		if err := decodeArguments(req.Arguments, &input); err != nil {
			return errorResponse(req.Tool, err), nil
		}
		result, err := s.engine.Sync(input)
		if err != nil {
			return errorResponse(req.Tool, err), nil
		}
		return okResponse(req.Tool, result), nil

	case ToolExplain:
		var input engine.ExplainInput
		if err := decodeArguments(req.Arguments, &input); err != nil {
			return errorResponse(req.Tool, err), nil
		}
		result, err := s.engine.Explain(input)
		if err != nil {
			return errorResponse(req.Tool, err), nil
		}
		return okResponse(req.Tool, result), nil

	default:
		return CallResponse{}, ErrUnknownTool
	}
}

func DefaultTools() map[ToolName]ToolDefinition {
	return map[ToolName]ToolDefinition{
		ToolRemember: {
			Name:        ToolRemember,
			Description: "Store a new cognitive memory in a Brainfile.",
			InputSchema: rememberSchema(),
		},
		ToolRecall: {
			Name:        ToolRecall,
			Description: "Recall relevant memories from a Brainfile.",
			InputSchema: recallSchema(),
		},
		ToolGetContext: {
			Name:        ToolGetContext,
			Description: "Build retrieval context from a Brainfile.",
			InputSchema: contextSchema(),
		},
		ToolSync: {
			Name:        ToolSync,
			Description: "Run the slow-path sync over pending memories.",
			InputSchema: syncSchema(),
		},
		ToolExplain: {
			Name:        ToolExplain,
			Description: "Explain a prior recall, dedup, or context decision.",
			InputSchema: explainSchema(),
		},
	}
}

func decodeArguments(arguments map[string]any, target any) error {
	payload, err := json.Marshal(arguments)
	if err != nil {
		return err
	}

	return json.Unmarshal(payload, target)
}

func okResponse(tool ToolName, content any) CallResponse {
	return CallResponse{
		Tool:    tool,
		OK:      true,
		Content: content,
	}
}

func errorResponse(tool ToolName, err error) CallResponse {
	return CallResponse{
		Tool:  tool,
		OK:    false,
		Error: err.Error(),
	}
}
