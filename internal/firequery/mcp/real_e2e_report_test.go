package mcp_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/phmotad/firememory/internal/brainfile"
	fqmcp "github.com/phmotad/firememory/internal/firequery/mcp"
	"github.com/phmotad/firememory/internal/firequeryapp"
)

func TestRealMCPFlowGeneratesMarkdownReport(t *testing.T) {
	reportPath := filepath.Clean(filepath.Join("..", "..", "..", "docs", "reports", "firequery-real-mcp-e2e-report.md"))
	var report strings.Builder
	failures := make([]string, 0)

	defer func() {
		if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
			t.Fatalf("create reports dir: %v", err)
		}
		if err := os.WriteFile(reportPath, []byte(report.String()), 0o644); err != nil {
			t.Fatalf("write report: %v", err)
		}
	}()

	writeReportHeader(&report, "FireQuery Real MCP E2E Report")
	fmt.Fprintf(&report, "Generated at: `%s`\n\n", time.Now().UTC().Format(time.RFC3339))

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	distDir := filepath.Join(root, "dist")
	binaryPath := filepath.Join(distDir, "firequery.exe")
	modelsDir := filepath.Join(distDir, "models")

	fmt.Fprintf(&report, "## Environment\n\n")
	fmt.Fprintf(&report, "- binary: `%s`\n", binaryPath)
	fmt.Fprintf(&report, "- models: `%s`\n", modelsDir)
	fmt.Fprintf(&report, "- dist: `%s`\n\n", distDir)

	if _, err := os.Stat(binaryPath); err != nil {
		fmt.Fprintf(&report, "Binary not found: `%v`\n", err)
		t.Skipf("firequery binary not found (run: make build): %v", err)
	}
	if _, err := os.Stat(modelsDir); err != nil {
		fmt.Fprintf(&report, "Models dir not found: `%v`\n", err)
		t.Skipf("models not downloaded (run: fquery models pull): %v", err)
	}

	envValues := map[string]string{
		"FIREQUERY_REQUIRE_REAL_MODELS": "1",
		"FIREMEMORY_MODELS_DIR":         modelsDir,
		"FIREQUERY_SIMILARITY_MODEL":    "intfloat/multilingual-e5-small",
		"FIREQUERY_ENTITY_MODEL":        "gliner2-small",
		"FIREQUERY_INTENT_MODEL":        "microsoft/deberta-v3-small",
		"FIREQUERY_TRIGGER_MODEL":       "microsoft/deberta-v3-small",
	}

	fmt.Fprintf(&report, "## Model Configuration\n\n")
	fmt.Fprintf(&report, "| Key | Value |\n")
	fmt.Fprintf(&report, "|-----|-------|\n")
	for _, key := range []string{
		"FIREQUERY_REQUIRE_REAL_MODELS",
		"FIREMEMORY_MODELS_DIR",
		"FIREQUERY_SIMILARITY_MODEL",
		"FIREQUERY_ENTITY_MODEL",
		"FIREQUERY_INTENT_MODEL",
		"FIREQUERY_TRIGGER_MODEL",
	} {
		fmt.Fprintf(&report, "| %s | `%s` |\n", key, escapeTable(envValues[key]))
	}

	lookupEnv := func(key string) string { return envValues[key] }
	manager := firequeryapp.BuildRuntimeManager(lookupEnv)
	health, err := manager.Health(context.Background())
	if err != nil {
		fmt.Fprintf(&report, "\nRuntime health failed: `%v`\n", err)
		t.Fatalf("runtime health: %v", err)
	}

	fmt.Fprintf(&report, "\n## Runtime Health\n\n")
	fmt.Fprintf(&report, "- ready: `%t`\n", health.Ready)
	fmt.Fprintf(&report, "- backend: `%s`\n", health.Backend)
	fmt.Fprintf(&report, "- notes: `%s`\n\n", strings.Join(health.Notes, " | "))
	fmt.Fprintf(&report, "| Model | Healthy | Loaded | Note |\n")
	fmt.Fprintf(&report, "|-------|---------|--------|------|\n")
	for _, model := range health.Models {
		note := ""
		if len(model.Notes) > 0 {
			note = model.Notes[0]
		}
		fmt.Fprintf(&report, "| `%s` | `%t` | `%t` | %s |\n", model.ID, model.Healthy, model.Loaded, escapeTable(note))
	}

	brainPath := filepath.Join(t.TempDir(), "agent.fbrain")
	handle, err := brainfile.Create(brainPath, brainfile.CreateOptions{})
	if err != nil {
		fmt.Fprintf(&report, "\nBrainfile create failed: `%v`\n", err)
		t.Fatalf("brainfile create: %v", err)
	}
	if err := handle.Close(); err != nil {
		fmt.Fprintf(&report, "\nBrainfile close failed: `%v`\n", err)
		t.Fatalf("brainfile close: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = distDir
	cmd.Env = append(os.Environ(), mapToEnv(envValues)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(&report, "\nProcess start failed: `%v`\n", err)
		t.Fatalf("process start: %v", err)
	}

	reader := bufio.NewReader(stdout)

	assertReport := func(name string, ok bool, detail string) {
		status := "PASS"
		if !ok {
			status = "FAIL"
			failures = append(failures, fmt.Sprintf("%s: %s", name, detail))
		}
		fmt.Fprintf(&report, "| %s | %s | %s |\n", name, status, escapeTable(detail))
	}

	fmt.Fprintf(&report, "\n## Assertions\n\n")
	fmt.Fprintf(&report, "| Check | Status | Detail |\n")
	fmt.Fprintf(&report, "|-------|--------|--------|\n")

	initialize := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
		},
	}
	if err := writeRPCRequest(stdin, initialize); err != nil {
		t.Fatalf("write initialize: %v", err)
	}
	initResponse := readRPCMap(t, reader)
	initResult := decodeMap(t, initResponse["result"])
	assertReport("initialize protocol", initResult["protocolVersion"] == "2024-11-05", fmt.Sprintf("protocol=%v", initResult["protocolVersion"]))

	if err := writeRPCRequest(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}); err != nil {
		t.Fatalf("write tools/list: %v", err)
	}
	toolsResponse := readRPCMap(t, reader)
	toolsResult := decodeMap(t, toolsResponse["result"])
	tools, _ := toolsResult["tools"].([]any)
	assertReport("tools list count", len(tools) == 6, fmt.Sprintf("tool_count=%d", len(tools)))

	remember1 := callTool(t, stdin, reader, fqmcp.ToolRemember, 3, map[string]any{
		"version":    "0.1",
		"request_id": "req_remember_1",
		"language":   "en",
		"actor": map[string]any{
			"type": "agent",
			"id":   "cursor",
		},
		"brain": brainPath,
		"input": map[string]any{
			"content":     "Joao uses Firebird 2.5 and reported a fiscal NF-e error after update 3.2",
			"allow_write": true,
		},
	})
	remember2 := callTool(t, stdin, reader, fqmcp.ToolRemember, 4, map[string]any{
		"version":    "0.1",
		"request_id": "req_remember_2",
		"language":   "en",
		"actor": map[string]any{
			"type": "agent",
			"id":   "cursor",
		},
		"brain": brainPath,
		"input": map[string]any{
			"content":     "Joao still uses Firebird 2.5 in the fiscal server and needs help with the invoice workflow",
			"allow_write": true,
		},
	})
	recall := callTool(t, stdin, reader, fqmcp.ToolRecall, 5, map[string]any{
		"version":    "0.1",
		"request_id": "req_recall",
		"language":   "en",
		"actor": map[string]any{
			"type": "agent",
			"id":   "cursor",
		},
		"brain": brainPath,
		"input": map[string]any{
			"query": "Does Joao use Firebird 2.5?",
			"top_k": 3,
		},
	})
	contextResult := callTool(t, stdin, reader, fqmcp.ToolGetContext, 6, map[string]any{
		"version":    "0.1",
		"request_id": "req_context",
		"language":   "en",
		"actor": map[string]any{
			"type": "agent",
			"id":   "cursor",
		},
		"brain": brainPath,
		"input": map[string]any{
			"task":          "Answer whether Joao still uses Firebird 2.5 and mention the fiscal issue after update 3.2",
			"budget_tokens": 1200,
		},
	})

	_ = stdin.Close()
	waitErr := cmd.Wait()
	if waitErr != nil {
		fmt.Fprintf(&report, "\nProcess wait error: `%v`\n", waitErr)
		t.Fatalf("process wait: %v", waitErr)
	}

	remember1Content := structuredContent(t, remember1)
	remember2Content := structuredContent(t, remember2)
	recallContent := structuredContent(t, recall)
	contextContent := structuredContent(t, contextResult)

	assertReport("runtime ready", health.Ready, strings.Join(health.Notes, " | "))
	assertReport("onnx backend active", containsRuntimeNote(health.Notes, "onnx backend"), strings.Join(health.Notes, " | "))
	assertReport("remember #1 ok", boolValue(remember1Content["ok"]), fmt.Sprintf("operation=%v", remember1Content["operation"]))
	assertReport("remember #1 intent", stringValueAny(remember1Content["data"], "intent") == "remember_information", fmt.Sprintf("intent=%s", stringValueAny(remember1Content["data"], "intent")))
	assertReport("remember #1 entities", numberValueAny(remember1Content["data"], "entity_count") > 0, fmt.Sprintf("entity_count=%.0f", numberValueAny(remember1Content["data"], "entity_count")))
	assertReport("remember #2 ok", boolValue(remember2Content["ok"]), fmt.Sprintf("operation=%v", remember2Content["operation"]))
	assertReport("recall ok", boolValue(recallContent["ok"]), fmt.Sprintf("operation=%v", recallContent["operation"]))
	assertReport("recall finds Firebird", recallContains(recallContent, "Firebird 2.5"), summarizeRecallHits(recallContent))
	assertReport("context ok", boolValue(contextContent["ok"]), fmt.Sprintf("operation=%v", contextContent["operation"]))
	assertReport("context mentions Joao", strings.Contains(stringValueAny(contextContent["data"], "context"), "Joao"), trimForReport(stringValueAny(contextContent["data"], "context"), 220))
	assertReport("context mentions Firebird", strings.Contains(stringValueAny(contextContent["data"], "context"), "Firebird 2.5"), trimForReport(stringValueAny(contextContent["data"], "context"), 220))
	assertReport("context mentions fiscal issue", strings.Contains(strings.ToLower(stringValueAny(contextContent["data"], "context")), "fiscal"), trimForReport(stringValueAny(contextContent["data"], "context"), 220))
	assertReport("pipeline trace present", hasPipelineTrace(contextContent), summarizeTraceKeys(contextContent))

	fmt.Fprintf(&report, "\n## MCP Responses\n\n")
	writeStructuredResponse(&report, "Remember #1", remember1Content)
	writeStructuredResponse(&report, "Remember #2", remember2Content)
	writeStructuredResponse(&report, "Recall", recallContent)
	writeStructuredResponse(&report, "Get Context", contextContent)

	if stderr.Len() > 0 {
		fmt.Fprintf(&report, "\n## Process STDERR Summary\n\n")
		fmt.Fprintf(&report, "```text\n%s\n```\n", trimForReport(stderr.String(), 4000))
	}

	fmt.Fprintf(&report, "\n## Verdict\n\n")
	if len(failures) == 0 {
		fmt.Fprintf(&report, "All assertions passed. The FireQuery MCP server started under `FIREQUERY_REQUIRE_REAL_MODELS=1`, accepted English-only MCP requests over `stdio`, persisted memories, recalled them, and built context using the ONNX model stack (`multilingual-e5-small`, `gliner-small-v2.1`, `deberta-v3-small`).\n")
	} else {
		for _, failure := range failures {
			fmt.Fprintf(&report, "- %s\n", failure)
		}
	}

	if len(failures) > 0 {
		t.Fatalf("real mcp flow failures: %s", strings.Join(failures, "; "))
	}
}

func writeRPCRequest(stdin io.Writer, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdin, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return err
}

func readRPCMap(t *testing.T, reader *bufio.Reader) map[string]any {
	t.Helper()

	message, err := readRPCMessageLocal(reader)
	if err != nil {
		t.Fatalf("readRPCMessage() error = %v", err)
	}
	raw, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Marshal(message) error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal(message) error = %v", err)
	}
	return decoded
}

func callTool(t *testing.T, stdin io.Writer, reader *bufio.Reader, tool string, id int, arguments map[string]any) map[string]any {
	t.Helper()

	if err := writeRPCRequest(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      tool,
			"arguments": arguments,
		},
	}); err != nil {
		t.Fatalf("write tool %s: %v", tool, err)
	}
	return readRPCMap(t, reader)
}

func structuredContent(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	result := decodeMap(t, response["result"])
	return decodeMap(t, result["structuredContent"])
}

func containsRuntimeNote(notes []string, want string) bool {
	for _, note := range notes {
		if strings.Contains(note, want) {
			return true
		}
	}
	return false
}

func boolValue(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func stringValueAny(container any, key string) string {
	mapped, ok := container.(map[string]any)
	if !ok {
		return ""
	}
	value, ok := mapped[key]
	if !ok {
		return ""
	}
	typed, ok := value.(string)
	if !ok {
		return ""
	}
	return typed
}

func numberValueAny(container any, key string) float64 {
	mapped, ok := container.(map[string]any)
	if !ok {
		return 0
	}
	value, ok := mapped[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	default:
		return 0
	}
}

func recallContains(content map[string]any, fragment string) bool {
	data, ok := content["data"].(map[string]any)
	if !ok {
		return false
	}
	rawHits, ok := data["hits"].([]any)
	if !ok {
		return false
	}
	for _, rawHit := range rawHits {
		hit, ok := rawHit.(map[string]any)
		if !ok {
			continue
		}
		text, _ := hit["content"].(string)
		if strings.Contains(text, fragment) {
			return true
		}
	}
	return false
}

func summarizeRecallHits(content map[string]any) string {
	data, ok := content["data"].(map[string]any)
	if !ok {
		return ""
	}
	rawHits, ok := data["hits"].([]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(rawHits))
	for _, rawHit := range rawHits {
		hit, ok := rawHit.(map[string]any)
		if !ok {
			continue
		}
		text, _ := hit["content"].(string)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " || ")
}

func hasPipelineTrace(content map[string]any) bool {
	trace, ok := content["trace"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = trace["pipeline"]
	return ok
}

func summarizeTraceKeys(content map[string]any) string {
	trace, ok := content["trace"].(map[string]any)
	if !ok {
		return ""
	}
	keys := make([]string, 0, len(trace))
	for key := range trace {
		keys = append(keys, key)
	}
	return strings.Join(keys, ", ")
}

func writeStructuredResponse(report *strings.Builder, title string, content map[string]any) {
	fmt.Fprintf(report, "### %s\n\n", title)
	data, _ := json.MarshalIndent(content, "", "  ")
	fmt.Fprintf(report, "```json\n%s\n```\n\n", data)
}

func mapToEnv(values map[string]string) []string {
	env := make([]string, 0, len(values))
	for key, value := range values {
		env = append(env, key+"="+value)
	}
	return env
}

type rpcMessageLocal struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   any             `json:"error,omitempty"`
}

func readRPCMessageLocal(reader *bufio.Reader) (rpcMessageLocal, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return rpcMessageLocal{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return rpcMessageLocal{}, fmt.Errorf("invalid mcp header: %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			length, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return rpcMessageLocal{}, err
			}
			contentLength = length
		}
	}
	if contentLength < 0 {
		return rpcMessageLocal{}, fmt.Errorf("missing Content-Length header")
	}
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return rpcMessageLocal{}, err
	}
	var message rpcMessageLocal
	if err := json.Unmarshal(payload, &message); err != nil {
		return rpcMessageLocal{}, err
	}
	return message, nil
}

func decodeMap(t *testing.T, value any) map[string]any {
	t.Helper()
	if typed, ok := value.(map[string]any); ok {
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

func writeReportHeader(report *strings.Builder, title string) {
	fmt.Fprintf(report, "# %s\n\n", title)
}

func trimForReport(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

func escapeTable(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "|", "\\|")
	return text
}
