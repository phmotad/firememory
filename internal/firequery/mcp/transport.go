package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

const defaultProtocolVersion = "2024-11-05"

type StdioServer struct {
	server          *Server
	serverName      string
	serverVersion   string
	protocolVersion string
}

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type toolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func NewStdioServer(server *Server, serverName, serverVersion string) *StdioServer {
	return &StdioServer{
		server:          server,
		serverName:      serverName,
		serverVersion:   serverVersion,
		protocolVersion: defaultProtocolVersion,
	}
}

func (s *StdioServer) Serve(ctx context.Context, reader io.Reader, writer io.Writer) error {
	buffered := bufio.NewReader(reader)
	for {
		message, err := readRPCMessage(buffered)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		response, ok := s.handleMessage(ctx, message)
		if !ok {
			continue
		}

		if err := writeRPCMessage(writer, response); err != nil {
			return err
		}
	}
}

func (s *StdioServer) handleMessage(ctx context.Context, message rpcMessage) (rpcMessage, bool) {
	switch message.Method {
	case "initialize":
		return s.handleInitialize(message), true
	case "notifications/initialized":
		return rpcMessage{}, false
	case "ping":
		return okRPC(message.ID, map[string]any{}), true
	case "tools/list":
		return okRPC(message.ID, map[string]any{
			"tools": s.listTools(),
		}), true
	case "tools/call":
		return s.handleToolsCall(ctx, message), true
	default:
		return errorRPC(message.ID, -32601, "method not found"), true
	}
}

func (s *StdioServer) handleInitialize(message rpcMessage) rpcMessage {
	params := initializeParams{}
	if len(message.Params) > 0 {
		_ = json.Unmarshal(message.Params, &params)
	}

	protocolVersion := strings.TrimSpace(params.ProtocolVersion)
	if protocolVersion == "" {
		protocolVersion = s.protocolVersion
	}

	return okRPC(message.ID, map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":    s.serverName,
			"version": s.serverVersion,
		},
	})
}

func (s *StdioServer) handleToolsCall(ctx context.Context, message rpcMessage) rpcMessage {
	var params toolsCallParams
	if err := json.Unmarshal(message.Params, &params); err != nil {
		return errorRPC(message.ID, -32602, "invalid tools/call params")
	}

	var request contract.ExternalRequest
	if err := decodeArguments(params.Arguments, &request); err != nil {
		return errorRPC(message.ID, -32602, "invalid tool arguments")
	}

	response, err := s.server.Handle(ctx, params.Name, request)
	if err != nil {
		return okRPC(message.ID, toolErrorResult(map[string]any{
			"ok":         false,
			"request_id": request.RequestID,
			"operation":  request.Operation,
			"rejected":   true,
			"error": map[string]any{
				"code":    "MCP_TOOL_ERROR",
				"message": err.Error(),
			},
		}))
	}

	return okRPC(message.ID, toolResult(response))
}

func (s *StdioServer) listTools() []map[string]any {
	schemas := DefaultSchemas()
	tools := make([]map[string]any, 0, len(schemas))
	for _, name := range []string{
		ToolAsk,
		ToolPlan,
		ToolRemember,
		ToolRecall,
		ToolGetContext,
		ToolExplain,
	} {
		schema := schemas[name]
		tools = append(tools, map[string]any{
			"name":        schema.Name,
			"description": schema.Description,
			"inputSchema": fireQueryToolInputSchema(name),
		})
	}
	return tools
}

func fireQueryToolInputSchema(name string) map[string]any {
	inputProperties := map[string]any{}
	requiredInput := []string{}

	switch name {
	case ToolRemember:
		inputProperties["content"] = map[string]any{"type": "string", "description": "Content to store in memory."}
		requiredInput = append(requiredInput, "content")
	case ToolRecall:
		inputProperties["query"] = map[string]any{"type": "string", "description": "Memory query text."}
		inputProperties["top_k"] = map[string]any{"type": "integer", "description": "Maximum number of hits."}
		requiredInput = append(requiredInput, "query")
	case ToolGetContext, ToolPlan:
		inputProperties["task"] = map[string]any{"type": "string", "description": "Task or response target."}
		inputProperties["budget_tokens"] = map[string]any{"type": "integer", "description": "Context token budget."}
		requiredInput = append(requiredInput, "task")
	case ToolExplain:
		inputProperties["target_operation"] = map[string]any{"type": "string", "description": "Operation to explain."}
		inputProperties["memory_id"] = map[string]any{"type": "string", "description": "Optional target memory id."}
		requiredInput = append(requiredInput, "target_operation")
	default:
		inputProperties["task"] = map[string]any{"type": "string", "description": "General request text."}
		inputProperties["content"] = map[string]any{"type": "string", "description": "Optional content to store."}
		inputProperties["query"] = map[string]any{"type": "string", "description": "Optional query text."}
	}

	return map[string]any{
		"type": "object",
		"required": []string{
			"version",
			"request_id",
			"actor",
			"brain",
			"input",
		},
		"properties": map[string]any{
			"version":    map[string]any{"type": "string"},
			"request_id": map[string]any{"type": "string"},
			"language":   map[string]any{"type": "string"},
			"actor": map[string]any{
				"type":     "object",
				"required": []string{"type", "id"},
				"properties": map[string]any{
					"type": map[string]any{"type": "string"},
					"id":   map[string]any{"type": "string"},
				},
			},
			"brain":     map[string]any{"type": "string"},
			"operation": map[string]any{"type": "string"},
			"input": map[string]any{
				"type":       "object",
				"properties": inputProperties,
				"required":   requiredInput,
			},
		},
	}
}

func toolResult(response contract.ExternalResponse) map[string]any {
	payload := map[string]any{
		"structuredContent": response,
		"isError":           !response.OK,
	}

	text, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		text = []byte(`{"ok":false,"error":{"code":"MCP_ENCODING_FAILED","message":"failed to encode response"}}`)
		payload["isError"] = true
	}

	payload["content"] = []map[string]any{
		{
			"type": "text",
			"text": string(text),
		},
	}
	return payload
}

func toolErrorResult(response map[string]any) map[string]any {
	text, _ := json.MarshalIndent(response, "", "  ")
	return map[string]any{
		"structuredContent": response,
		"isError":           true,
		"content": []map[string]any{
			{
				"type": "text",
				"text": string(text),
			},
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

func readRPCMessage(reader *bufio.Reader) (rpcMessage, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && contentLength == -1 {
				return rpcMessage{}, io.EOF
			}
			return rpcMessage{}, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return rpcMessage{}, fmt.Errorf("invalid mcp header: %q", line)
		}

		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			length, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return rpcMessage{}, fmt.Errorf("invalid content length: %w", err)
			}
			contentLength = length
		}
	}

	if contentLength < 0 {
		return rpcMessage{}, fmt.Errorf("missing Content-Length header")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return rpcMessage{}, err
	}

	var message rpcMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return rpcMessage{}, err
	}
	return message, nil
}

func writeRPCMessage(writer io.Writer, message rpcMessage) error {
	if message.JSONRPC == "" {
		message.JSONRPC = "2.0"
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "Content-Length: %d\r\n\r\n", len(payload))
	buffer.Write(payload)

	_, err = writer.Write(buffer.Bytes())
	return err
}

func okRPC(id json.RawMessage, result any) rpcMessage {
	return rpcMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func errorRPC(id json.RawMessage, code int, message string) rpcMessage {
	return rpcMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
		},
	}
}
