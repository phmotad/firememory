package storage

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func TestBboltStoreConcurrentReadWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent.fbrain")

	store, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	const workers = 8
	const writesPerWorker = 25

	var wg sync.WaitGroup
	errs := make(chan error, workers*2)

	for worker := range workers {
		worker := worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range writesPerWorker {
				key := fmt.Sprintf("worker_%02d_%03d", worker, i)
				if err := store.Put("memories", key, []byte(key)); err != nil {
					errs <- err
					return
				}
			}
		}()
	}

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < writesPerWorker; i++ {
				_, err := store.List("memories", "worker_", 0)
				if err != nil {
					errs <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent operation error: %v", err)
		}
	}

	records, err := store.List("memories", "worker_", 0)
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	if len(records) != workers*writesPerWorker {
		t.Fatalf("len(records) = %d, want %d", len(records), workers*writesPerWorker)
	}
}

func TestBboltStoreRepeatedReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen-loop.fbrain")

	for i := range 20 {
		store, err := OpenBboltStore(path)
		if err != nil {
			t.Fatalf("open iteration %d: %v", i, err)
		}

		key := fmt.Sprintf("mem_%02d", i)
		if err := store.Put("memories", key, []byte("value")); err != nil {
			t.Fatalf("put iteration %d: %v", i, err)
		}

		if err := store.Close(); err != nil {
			t.Fatalf("close iteration %d: %v", i, err)
		}
	}

	store, err := OpenBboltStore(path)
	if err != nil {
		t.Fatalf("final reopen: %v", err)
	}
	defer store.Close()

	records, err := store.List("memories", "mem_", 0)
	if err != nil {
		t.Fatalf("list after reopen loop: %v", err)
	}
	if len(records) != 20 {
		t.Fatalf("len(records) = %d, want 20", len(records))
	}
}
