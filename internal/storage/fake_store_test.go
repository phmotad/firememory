package storage

import (
	"bytes"
	"testing"
)

func TestFakeStorePutGetDelete(t *testing.T) {
	store := NewFakeStore()

	if err := store.Put("memories", "mem_01", []byte("alpha")); err != nil {
		t.Fatalf("expected put to succeed, got %v", err)
	}

	value, err := store.Get("memories", "mem_01")
	if err != nil {
		t.Fatalf("expected get to succeed, got %v", err)
	}

	if string(value) != "alpha" {
		t.Fatalf("expected value %q, got %q", "alpha", string(value))
	}

	if err := store.Delete("memories", "mem_01"); err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}

	if _, err := store.Get("memories", "mem_01"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestFakeStoreEnsureNamespace(t *testing.T) {
	store := NewFakeStore()

	if err := store.EnsureNamespace("memories"); err != nil {
		t.Fatalf("expected ensure namespace to succeed, got %v", err)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatalf("expected snapshot to succeed, got %v", err)
	}

	records, ok := snapshot.Namespaces["memories"]
	if !ok {
		t.Fatal("expected namespace to exist in snapshot")
	}

	if len(records) != 0 {
		t.Fatalf("expected empty namespace, got %d records", len(records))
	}
}

func TestFakeStoreListUsesPrefixAndLimit(t *testing.T) {
	store := NewFakeStore()

	for key, value := range map[string]string{
		"mem_03":  "three",
		"mem_01":  "one",
		"meta_01": "skip",
		"mem_02":  "two",
	} {
		if err := store.Put("memories", key, []byte(value)); err != nil {
			t.Fatalf("put %q failed: %v", key, err)
		}
	}

	records, err := store.List("memories", "mem_", 2)
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	if records[0].Key != "mem_01" || records[1].Key != "mem_02" {
		t.Fatalf("expected sorted keys [mem_01 mem_02], got [%s %s]", records[0].Key, records[1].Key)
	}
}

func TestFakeStoreViewIsReadOnly(t *testing.T) {
	store := NewFakeStore()

	err := store.View(func(tx Tx) error {
		return tx.Put("memories", "mem_01", []byte("alpha"))
	})

	if err != ErrReadOnlyTx {
		t.Fatalf("expected ErrReadOnlyTx, got %v", err)
	}
}

func TestFakeStoreSnapshotClonesData(t *testing.T) {
	store := NewFakeStore()
	payload := []byte("alpha")

	if err := store.Put("memories", "mem_01", payload); err != nil {
		t.Fatalf("expected put to succeed, got %v", err)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatalf("expected snapshot to succeed, got %v", err)
	}

	payload[0] = 'z'

	record := snapshot.Namespaces["memories"][0]
	if !bytes.Equal(record.Value, []byte("alpha")) {
		t.Fatalf("expected snapshot to keep original bytes, got %q", string(record.Value))
	}
}

func TestFakeStoreCloseRejectsFurtherOperations(t *testing.T) {
	store := NewFakeStore()

	if err := store.Close(); err != nil {
		t.Fatalf("expected close to succeed, got %v", err)
	}

	if err := store.Put("memories", "mem_01", []byte("alpha")); err != ErrStoreClosed {
		t.Fatalf("expected ErrStoreClosed, got %v", err)
	}
}
