package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/phmotad/firememory/internal/brainfile"
)

func TestErrorDiagnostic(t *testing.T) {
	t.Parallel()

	info := ErrorDiagnostic(brainfile.ErrInvalidExtension)
	if info.Code != "FMEM_INVALID_EXTENSION" {
		t.Fatalf("Code = %q, want FMEM_INVALID_EXTENSION", info.Code)
	}
	if info.Message == "" {
		t.Fatal("expected message")
	}
}

func TestExtractJSONFlag(t *testing.T) {
	t.Parallel()

	args, jsonOutput := ExtractJSONFlag([]string{"inspect", "--json", "agent.fbrain"})
	if !jsonOutput {
		t.Fatal("expected jsonOutput=true")
	}
	if len(args) != 2 || args[0] != "inspect" || args[1] != "agent.fbrain" {
		t.Fatalf("args = %#v", args)
	}
}

func TestJSONLogger(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := NewJSONLogger(&buffer)
	if err := logger.Log(LogEvent{
		Level:     "info",
		Component: "test",
		Message:   "hello",
		Fields:    map[string]any{"ok": true},
	}); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["component"] != "test" {
		t.Fatalf("component = %#v, want test", decoded["component"])
	}
}

func TestFallbackErrorCode(t *testing.T) {
	t.Parallel()

	if code := ErrorCode(errors.New("boom")); code != "INTERNAL_ERROR" {
		t.Fatalf("code = %q, want INTERNAL_ERROR", code)
	}
}
