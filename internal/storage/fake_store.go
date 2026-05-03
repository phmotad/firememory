package storage

import (
	"bytes"
	"sort"
	"strings"
	"sync"
	"time"
)

type FakeStore struct {
	mu     sync.RWMutex
	closed bool
	data   map[string]map[string][]byte
}

func NewFakeStore() *FakeStore {
	return &FakeStore{
		data: map[string]map[string][]byte{},
	}
}

func (s *FakeStore) EnsureNamespace(namespace string) error {
	return s.Update(func(tx Tx) error {
		return tx.EnsureNamespace(namespace)
	})
}

func (s *FakeStore) Put(namespace, key string, value []byte) error {
	return s.Update(func(tx Tx) error {
		return tx.Put(namespace, key, value)
	})
}

func (s *FakeStore) Get(namespace, key string) ([]byte, error) {
	var out []byte

	err := s.View(func(tx Tx) error {
		value, err := tx.Get(namespace, key)
		if err != nil {
			return err
		}

		out = value
		return nil
	})

	return out, err
}

func (s *FakeStore) Delete(namespace, key string) error {
	return s.Update(func(tx Tx) error {
		return tx.Delete(namespace, key)
	})
}

func (s *FakeStore) List(namespace, prefix string, limit int) ([]Record, error) {
	var out []Record

	err := s.View(func(tx Tx) error {
		records, err := tx.List(namespace, prefix, limit)
		if err != nil {
			return err
		}

		out = records
		return nil
	})

	return out, err
}

func (s *FakeStore) View(fn func(Tx) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return ErrStoreClosed
	}

	return fn(&fakeTx{
		store:    s,
		readOnly: true,
	})
}

func (s *FakeStore) Update(fn func(Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStoreClosed
	}

	return fn(&fakeTx{
		store:    s,
		readOnly: false,
	})
}

func (s *FakeStore) Snapshot() (Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return Snapshot{}, ErrStoreClosed
	}

	namespaces := make(map[string][]Record, len(s.data))
	for namespace, items := range s.data {
		records := make([]Record, 0, len(items))
		for key, value := range items {
			records = append(records, Record{
				Key:   key,
				Value: cloneBytes(value),
			})
		}

		sort.Slice(records, func(i, j int) bool {
			return records[i].Key < records[j].Key
		})

		namespaces[namespace] = records
	}

	return Snapshot{
		TakenAt:    time.Now().UTC(),
		Namespaces: namespaces,
	}, nil
}

func (s *FakeStore) Compact() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return ErrStoreClosed
	}

	return nil
}

func (s *FakeStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	return nil
}

type fakeTx struct {
	store    *FakeStore
	readOnly bool
}

func (tx *fakeTx) EnsureNamespace(namespace string) error {
	if tx.readOnly {
		return ErrReadOnlyTx
	}

	if strings.TrimSpace(namespace) == "" {
		return ErrNamespaceRequired
	}

	if tx.store.data[namespace] == nil {
		tx.store.data[namespace] = map[string][]byte{}
	}

	return nil
}

func (tx *fakeTx) Put(namespace, key string, value []byte) error {
	if tx.readOnly {
		return ErrReadOnlyTx
	}

	if err := validateNamespaceKey(namespace, key); err != nil {
		return err
	}

	if value == nil {
		return ErrValueRequired
	}

	if err := tx.EnsureNamespace(namespace); err != nil {
		return err
	}

	tx.store.data[namespace][key] = cloneBytes(value)
	return nil
}

func (tx *fakeTx) Get(namespace, key string) ([]byte, error) {
	if err := validateNamespaceKey(namespace, key); err != nil {
		return nil, err
	}

	namespaceData := tx.store.data[namespace]
	if namespaceData == nil {
		return nil, ErrNotFound
	}

	value, ok := namespaceData[key]
	if !ok {
		return nil, ErrNotFound
	}

	return cloneBytes(value), nil
}

func (tx *fakeTx) Delete(namespace, key string) error {
	if tx.readOnly {
		return ErrReadOnlyTx
	}

	if err := validateNamespaceKey(namespace, key); err != nil {
		return err
	}

	namespaceData := tx.store.data[namespace]
	if namespaceData == nil {
		return nil
	}

	delete(namespaceData, key)
	if len(namespaceData) == 0 {
		delete(tx.store.data, namespace)
	}

	return nil
}

func (tx *fakeTx) List(namespace, prefix string, limit int) ([]Record, error) {
	if strings.TrimSpace(namespace) == "" {
		return nil, ErrNamespaceRequired
	}

	if limit < 0 {
		return nil, ErrNegativeListLimit
	}

	namespaceData := tx.store.data[namespace]
	if namespaceData == nil {
		return nil, nil
	}

	keys := make([]string, 0, len(namespaceData))
	for key := range namespaceData {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)

	if limit > 0 && len(keys) > limit {
		keys = keys[:limit]
	}

	records := make([]Record, 0, len(keys))
	for _, key := range keys {
		records = append(records, Record{
			Key:   key,
			Value: cloneBytes(namespaceData[key]),
		})
	}

	return records, nil
}

func validateNamespaceKey(namespace, key string) error {
	if strings.TrimSpace(namespace) == "" {
		return ErrNamespaceRequired
	}

	if strings.TrimSpace(key) == "" {
		return ErrKeyRequired
	}

	return nil
}

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}

	return bytes.Clone(value)
}
