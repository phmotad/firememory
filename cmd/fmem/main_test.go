package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIInitRememberRecallSyncContextInspectSnapshotBackupRestoreCompact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	out := runCLI(t, "init", path)
	assertContains(t, out, "initialized brainfile")

	out = runCLI(t, "remember", path, "Cliente", "Joao", "usa", "Firebird", "2.5", "e", "teve", "erro", "fiscal", "na", "NF-e", "apos", "atualizacao", "3.2")
	assertContains(t, out, "action: create_new")

	out = runCLI(t, "remember", path, "Joao", "relatou", "novamente", "problema", "fiscal", "na", "NF-e", "apos", "a", "versao", "3.2")
	if !strings.Contains(out, "action:") {
		t.Fatalf("expected remember output, got %q", out)
	}

	out = runCLI(t, "recall", path, "problema", "fiscal", "NF-e")
	assertContains(t, out, "1. [")

	out = runCLI(t, "sync", path)
	assertContains(t, out, "processed:")

	out = runCLI(t, "context", path, "responder", "Joao", "sobre", "erro", "fiscal", "apos", "atualizacao")
	assertContains(t, out, "Memories:")
	assertContains(t, out, "Estimated tokens:")

	out = runCLI(t, "inspect", path)
	assertContains(t, out, "brainfile:")
	assertContains(t, out, "namespaces:")

	out = runCLI(t, "snapshot", path)
	assertContains(t, out, "snapshot taken at:")

	backupPath := filepath.Join(t.TempDir(), "agent.backup")
	out = runCLI(t, "backup", path, backupPath)
	assertContains(t, out, "backup created:")

	restoredPath := filepath.Join(t.TempDir(), "restored-agent.fbrain")
	out = runCLI(t, "restore", backupPath, restoredPath)
	assertContains(t, out, "brainfile restored:")

	out = runCLI(t, "compact", path)
	assertContains(t, out, "compact complete")
}

func TestCLIAcceptanceFlowShowsPersistentUsableBrainfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "acceptance-agent.fbrain")

	initOut := runCLI(t, "init", path)
	assertContains(t, initOut, "initialized brainfile")

	rememberOne := runCLI(t, "remember", path, "Cliente", "Joao", "usa", "Firebird", "2.5", "e", "teve", "erro", "fiscal", "na", "NF-e", "apos", "atualizacao", "3.2")
	assertContains(t, rememberOne, "status: pending_sync")

	rememberTwo := runCLI(t, "remember", path, "Joao", "relatou", "novamente", "problema", "fiscal", "em", "nota", "eletronica", "depois", "da", "versao", "3.2")
	assertContains(t, rememberTwo, "memory_id:")

	recallOut := runCLI(t, "recall", path, "problema", "fiscal", "NF-e")
	assertContains(t, recallOut, "1. [")
	assertContains(t, recallOut, "erro fiscal")

	syncOut := runCLI(t, "sync", path)
	assertContains(t, syncOut, "processed: 2")

	contextOut := runCLI(t, "context", path, "responder", "Joao", "sobre", "erro", "fiscal", "apos", "atualizacao")
	assertContains(t, contextOut, "Memories:")
	assertContains(t, contextOut, "Estimated tokens:")

	inspectOut := runCLI(t, "inspect", path)
	assertContains(t, inspectOut, "brainfile:")
	assertContains(t, inspectOut, "- memories:")
	assertContains(t, inspectOut, "- vectors:")
}

func TestCLIUsageOnUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"unknown"}, &stdout, &stderr, false)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}

	if !strings.Contains(stderr.String(), "usage: fmem") {
		t.Fatalf("expected usage output, got %q", stderr.String())
	}
}

func runCLI(t *testing.T, args ...string) string {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run(args, &stdout, &stderr, false); err != nil {
		t.Fatalf("run(%v) error: %v, stderr=%q", args, err, stderr.String())
	}

	return stdout.String()
}

func assertContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q to contain %q", text, want)
	}
}

func TestCLIJSONInspect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")
	runCLI(t, "init", path)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"inspect", path}, &stdout, &stderr, true); err != nil {
		t.Fatalf("run(json inspect) error: %v, stderr=%q", err, stderr.String())
	}

	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["Path"] == nil && decoded["path"] == nil {
		t.Fatalf("decoded = %#v", decoded)
	}
}
