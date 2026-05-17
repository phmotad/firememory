package storage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenBboltStoreRequiresPath(t *testing.T) {
	store, err := OpenBboltStore("")
	if err != ErrPathRequired {
		t.Fatalf("expected ErrPathRequired, got %v", err)
	}

	if store != nil {
		t.Fatal("expected nil store on invalid path")
	}
}

func TestBboltStorePersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	store, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	if err := store.Put("memories", "mem_01", []byte("alpha")); err != nil {
		t.Fatalf("put: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer reopened.Close()

	value, err := reopened.Get("memories", "mem_01")
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}

	if string(value) != "alpha" {
		t.Fatalf("expected persisted value %q, got %q", "alpha", string(value))
	}
}

func TestBboltStoreEnsureNamespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	store, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.EnsureNamespace("memories"); err != nil {
		t.Fatalf("ensure namespace: %v", err)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	records, ok := snapshot.Namespaces["memories"]
	if !ok {
		t.Fatal("expected namespace to exist in snapshot")
	}

	if len(records) != 0 {
		t.Fatalf("expected empty namespace, got %d records", len(records))
	}
}

func TestBboltStoreListSnapshotAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	store, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	for key, value := range map[string]string{
		"mem_02":  "two",
		"mem_01":  "one",
		"meta_01": "skip",
	} {
		if err := store.Put("memories", key, []byte(value)); err != nil {
			t.Fatalf("put %q: %v", key, err)
		}
	}

	records, err := store.List("memories", "mem_", 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 prefixed records, got %d", len(records))
	}

	if records[0].Key != "mem_01" || records[1].Key != "mem_02" {
		t.Fatalf("expected sorted keys [mem_01 mem_02], got [%s %s]", records[0].Key, records[1].Key)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if len(snapshot.Namespaces["memories"]) != 3 {
		t.Fatalf("expected snapshot to contain 3 records, got %d", len(snapshot.Namespaces["memories"]))
	}

	records[0].Value[0] = 'z'
	if bytes.Equal(snapshot.Namespaces["memories"][0].Value, records[0].Value) {
		t.Fatal("expected snapshot bytes to be cloned")
	}

	if err := store.Delete("memories", "mem_01"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := store.Get("memories", "mem_01"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestBboltStoreViewIsReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	store, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	err = store.View(func(tx Tx) error {
		return tx.Put("memories", "mem_01", []byte("alpha"))
	})
	if err != ErrReadOnlyTx {
		t.Fatalf("expected ErrReadOnlyTx, got %v", err)
	}
}

func TestBboltStoreRejectsOperationsAfterClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	store, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if err := store.Put("memories", "mem_01", []byte("alpha")); err != ErrStoreClosed {
		t.Fatalf("expected ErrStoreClosed, got %v", err)
	}
}

func TestBboltStoreRejectsConcurrentOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	store, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// The explicit lock file was removed; bbolt uses its OS-level flock with a
	// 5-second timeout. A second concurrent open must fail. This test intentionally
	// waits up to boltOpenTimeout (~5s) before the error is returned by bbolt.
	second, err := OpenBboltStore(path)
	if err == nil {
		second.Close()
		t.Fatal("expected error on concurrent open, got nil")
	}
	if second != nil {
		t.Fatalf("expected nil second store on error, got %#v", second)
	}
}

func TestBboltStoreCompactKeepsDataAndRewritesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	store, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	for i := range 20 {
		key := fmt.Sprintf("mem_%02d", i)
		if err := store.Put("memories", key, bytes.Repeat([]byte("x"), 1024)); err != nil {
			t.Fatalf("put %q: %v", key, err)
		}
	}
	if err := store.Delete("memories", "mem_00"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat before compact: %v", err)
	}

	if err := store.Compact(); err != nil {
		t.Fatalf("compact: %v", err)
	}

	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after compact: %v", err)
	}
	if afterInfo.Size() <= 0 || beforeInfo.Size() <= 0 {
		t.Fatalf("invalid file sizes before=%d after=%d", beforeInfo.Size(), afterInfo.Size())
	}

	value, err := store.Get("memories", "mem_01")
	if err != nil {
		t.Fatalf("get after compact: %v", err)
	}
	if len(value) != 1024 {
		t.Fatalf("len(value) = %d, want 1024", len(value))
	}
}
