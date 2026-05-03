package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

func TestStdioServerInitializeAndToolsList(t *testing.T) {
	t.Parallel()

	server := NewServer()
	server.RegisterDefaultTools(func(_ context.Context, request contract.ExternalRequest) (contract.ExternalResponse, error) {
		return contract.ExternalResponse{
			OK:        true,
			RequestID: request.RequestID,
			Operation: request.Operation,
		}, nil
	})

	stdio := NewStdioServer(server, "firequery", "0.1.0")

	var input bytes.Buffer
	writeTestRPC(t, &input, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
		},
	})
	writeTestRPC(t, &input, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})

	var output bytes.Buffer
	if err := stdio.Serve(context.Background(), &input, &output); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	responses := readAllTestRPC(t, output.Bytes(), 2)
	first := responses[0]
	second := responses[1]

	firstResult := decodeResultMap(t, first["result"])
	if firstResult["protocolVersion"] != "2024-11-05" {
		t.Fatalf("protocolVersion = %#v", firstResult["protocolVersion"])
	}

	secondResult := decodeResultMap(t, second["result"])
	tools, ok := secondResult["tools"].([]any)
	if !ok {
		t.Fatalf("tools = %#v", secondResult["tools"])
	}
	if len(tools) != 6 {
		t.Fatalf("expected 6 tools, got %d", len(tools))
	}
}

func TestStdioServerToolsCall(t *testing.T) {
	t.Parallel()

	server := NewServer()
	server.RegisterDefaultTools(func(_ context.Context, request contract.ExternalRequest) (contract.ExternalResponse, error) {
		return contract.ExternalResponse{
			OK:        true,
			RequestID: request.RequestID,
			Operation: request.Operation,
			Data: map[string]any{
				"brain": request.Brain,
			},
		}, nil
	})

	stdio := NewStdioServer(server, "firequery", "0.1.0")

	var input bytes.Buffer
	writeTestRPC(t, &input, map[string]any{
		"jsonrpc": "2.0",
		"id":      "call-1",
		"method":  "tools/call",
		"params": map[string]any{
			"name": ToolRecall,
			"arguments": map[string]any{
				"version":    "0.1",
				"request_id": "req_1",
				"language":   "en",
				"actor": map[string]any{
					"type": "agent",
					"id":   "tester",
				},
				"brain": "agent.fbrain",
				"input": map[string]any{
					"query": "fiscal error",
				},
			},
		},
	})

	var output bytes.Buffer
	if err := stdio.Serve(context.Background(), &input, &output); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	response := readAllTestRPC(t, output.Bytes(), 1)[0]
	result := decodeResultMap(t, response["result"])
	if result["isError"] != false {
		t.Fatalf("isError = %#v", result["isError"])
	}

	structured := decodeResultMap(t, result["structuredContent"])
	if structured["operation"] != "recall" {
		t.Fatalf("operation = %#v", structured["operation"])
	}
	data := decodeResultMap(t, structured["data"])
	if data["brain"] != "agent.fbrain" {
		t.Fatalf("brain = %#v", data["brain"])
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content = %#v", result["content"])
	}
	contentItem := decodeResultMap(t, content[0])
	if !strings.Contains(contentItem["text"].(string), "\"operation\": \"recall\"") {
		t.Fatalf("text content = %#v", contentItem["text"])
	}
}

func writeTestRPC(t *testing.T, buffer *bytes.Buffer, payload map[string]any) {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	buffer.WriteString("Content-Length: ")
	buffer.WriteString(strconv.Itoa(len(data)))
	buffer.WriteString("\r\n\r\n")
	buffer.Write(data)
}

func readAllTestRPC(t *testing.T, payload []byte, count int) []map[string]any {
	t.Helper()

	reader := bufio.NewReader(bytes.NewReader(payload))
	results := make([]map[string]any, 0, count)
	for range count {
		message, err := readRPCMessage(reader)
		if err != nil {
			t.Fatalf("readRPCMessage() error = %v", err)
		}

		raw, err := json.Marshal(message)
		if err != nil {
			t.Fatalf("Marshal(message) error = %v", err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		results = append(results, decoded)
	}
	return results
}

func decodeResultMap(t *testing.T, value any) map[string]any {
	t.Helper()

	typed, ok := value.(map[string]any)
	if ok {
		return typed
	}

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	return decoded
}
