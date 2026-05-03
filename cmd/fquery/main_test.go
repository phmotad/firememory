package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunDevices(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"devices"}, &stdout, &stderr, func(key string) string {
		if key == "FIREQUERY_ENABLE_CUDA" {
			return "1"
		}
		return ""
	}, false)
	if err != nil {
		t.Fatalf("run(devices) error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "cpu available=true backend=cpu") {
		t.Fatalf("missing cpu line in %q", out)
	}
	if !strings.Contains(out, "cuda available=true backend=cuda") {
		t.Fatalf("missing cuda line in %q", out)
	}
}

func TestRunDoctor(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"doctor"}, &stdout, &stderr, func(string) string { return "" }, false)
	if err != nil {
		t.Fatalf("run(doctor) error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "ready: true") {
		t.Fatalf("missing ready line in %q", out)
	}
	if !strings.Contains(out, "runtime [ok] backend=cpu") {
		t.Fatalf("missing runtime line in %q", out)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"unknown"}, &stdout, &stderr, func(string) string { return "" }, false)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if !strings.Contains(stderr.String(), "usage: fquery <command>") {
		t.Fatalf("missing usage in %q", stderr.String())
	}
}

func TestRunDoctorJSON(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"doctor"}, &stdout, &stderr, func(string) string { return "" }, true)
	if err != nil {
		t.Fatalf("run(doctor --json) error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := decoded["Ready"]; !ok {
		if _, ok := decoded["ready"]; !ok {
			t.Fatalf("decoded = %#v", decoded)
		}
	}
}
