package mcp

import (
	"path/filepath"
	"testing"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/engine"
)

func TestServerExposesDefaultTools(t *testing.T) {
	tools := DefaultTools()
	if len(tools) != 5 {
		t.Fatalf("expected 5 tools, got %d", len(tools))
	}

	if _, ok := tools[ToolRemember]; !ok {
		t.Fatal("expected firememory.remember tool")
	}

	if _, ok := tools[ToolGetContext]; !ok {
		t.Fatal("expected firememory.get_context tool")
	}
}

func TestServerHandleCallRememberRecallSyncContextExplain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	handle, err := brainfile.Create(path, brainfile.CreateOptions{Name: "agent"})
	if err != nil {
		t.Fatalf("create brainfile: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close brainfile: %v", err)
	}

	eng, err := engine.Open(engine.Options{Path: path})
	if err != nil {
		t.Fatalf("open engine: %v", err)
	}
	defer eng.Close()

	server := NewServer(eng)

	rememberResp, err := server.HandleCall(CallRequest{
		Tool: ToolRemember,
		Arguments: map[string]any{
			"brain_path": path,
			"content":    "Cliente Joao usa Firebird 2.5 e teve erro fiscal na NF-e apos atualizacao 3.2",
		},
	})
	if err != nil {
		t.Fatalf("remember call: %v", err)
	}
	if !rememberResp.OK {
		t.Fatalf("expected remember OK, got error %q", rememberResp.Error)
	}

	recallResp, err := server.HandleCall(CallRequest{
		Tool: ToolRecall,
		Arguments: map[string]any{
			"brain_path": path,
			"query":      "erro fiscal NF-e",
			"top_k":      1,
		},
	})
	if err != nil {
		t.Fatalf("recall call: %v", err)
	}
	if !recallResp.OK {
		t.Fatalf("expected recall OK, got error %q", recallResp.Error)
	}

	syncResp, err := server.HandleCall(CallRequest{
		Tool: ToolSync,
		Arguments: map[string]any{
			"brain_path": path,
		},
	})
	if err != nil {
		t.Fatalf("sync call: %v", err)
	}
	if !syncResp.OK {
		t.Fatalf("expected sync OK, got error %q", syncResp.Error)
	}

	contextResp, err := server.HandleCall(CallRequest{
		Tool: ToolGetContext,
		Arguments: map[string]any{
			"brain_path":    path,
			"query":         "responder Joao sobre erro fiscal",
			"budget_tokens": 200,
			"include_graph": true,
		},
	})
	if err != nil {
		t.Fatalf("context call: %v", err)
	}
	if !contextResp.OK {
		t.Fatalf("expected context OK, got error %q", contextResp.Error)
	}

	explainResp, err := server.HandleCall(CallRequest{
		Tool: ToolExplain,
		Arguments: map[string]any{
			"brain_path": path,
			"operation":  "context",
			"trace": []string{
				"recalled relevant memories",
				"built context text within budget",
			},
		},
	})
	if err != nil {
		t.Fatalf("explain call: %v", err)
	}
	if !explainResp.OK {
		t.Fatalf("expected explain OK, got error %q", explainResp.Error)
	}
}

func TestServerRejectsUnknownTool(t *testing.T) {
	server := NewServer(nil)

	_, err := server.HandleCall(CallRequest{
		Tool: ToolName("firememory.unknown"),
	})
	if err != ErrUnknownTool {
		t.Fatalf("expected ErrUnknownTool, got %v", err)
	}
}

func TestServerReturnsStructuredErrorForBadArguments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "badargs.fbrain")

	handle, err := brainfile.Create(path, brainfile.CreateOptions{Name: "badargs"})
	if err != nil {
		t.Fatalf("create brainfile: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close brainfile: %v", err)
	}

	eng, err := engine.Open(engine.Options{Path: path})
	if err != nil {
		t.Fatalf("open engine: %v", err)
	}
	defer eng.Close()

	server := NewServer(eng)

	resp, err := server.HandleCall(CallRequest{
		Tool: ToolRemember,
		Arguments: map[string]any{
			"brain_path": path,
		},
	})
	if err != nil {
		t.Fatalf("remember call: %v", err)
	}

	if resp.OK {
		t.Fatal("expected structured error response for invalid arguments")
	}

	if resp.Error == "" {
		t.Fatal("expected error message in response")
	}
}
