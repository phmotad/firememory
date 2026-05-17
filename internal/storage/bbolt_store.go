package storage

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
	"os"
)

// boltOpenTimeout is the maximum time to wait for bbolt's OS-level file lock.
// The daemon architecture serialises writes through a single process, so
// contention only occurs briefly during daemon startup/shutdown transitions.
const boltOpenTimeout = 5 * time.Second

type BboltStore struct {
	mu     sync.RWMutex
	db     *bolt.DB
	path   string
	closed bool
}

func OpenBboltStore(path string) (*BboltStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrPathRequired
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: boltOpenTimeout})
	if err != nil {
		return nil, err
	}

	return &BboltStore{
		db:   db,
		path: path,
	}, nil
}

func (s *BboltStore) EnsureNamespace(namespace string) error {
	return s.Update(func(tx Tx) error {
		return tx.EnsureNamespace(namespace)
	})
}

func (s *BboltStore) Put(namespace, key string, value []byte) error {
	return s.Update(func(tx Tx) error {
		return tx.Put(namespace, key, value)
	})
}

func (s *BboltStore) Get(namespace, key string) ([]byte, error) {
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

func (s *BboltStore) Delete(namespace, key string) error {
	return s.Update(func(tx Tx) error {
		return tx.Delete(namespace, key)
	})
}

func (s *BboltStore) List(namespace, prefix string, limit int) ([]Record, error) {
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

func (s *BboltStore) View(fn func(Tx) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return ErrStoreClosed
	}

	return s.db.View(func(tx *bolt.Tx) error {
		return fn(&boltTx{
			tx:       tx,
			readOnly: true,
		})
	})
}

func (s *BboltStore) Update(fn func(Tx) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return ErrStoreClosed
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		return fn(&boltTx{
			tx:       tx,
			readOnly: false,
		})
	})
}

func (s *BboltStore) Snapshot() (Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return Snapshot{}, ErrStoreClosed
	}

	snapshot := Snapshot{
		TakenAt:    time.Now().UTC(),
		Namespaces: map[string][]Record{},
	}

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, bucket *bolt.Bucket) error {
			namespace := string(name)
			records := make([]Record, 0, bucket.Stats().KeyN)

			err := bucket.ForEach(func(key, value []byte) error {
				if value == nil {
					return nil
				}

				records = append(records, Record{
					Key:   string(key),
					Value: cloneBytes(value),
				})
				return nil
			})
			if err != nil {
				return err
			}

			sort.Slice(records, func(i, j int) bool {
				return records[i].Key < records[j].Key
			})

			snapshot.Namespaces[namespace] = records
			return nil
		})
	})
	if err != nil {
		return Snapshot{}, err
	}

	return snapshot, nil
}

func (s *BboltStore) Compact() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStoreClosed
	}

	tempPath := s.path + ".compact.tmp"
	backupPath := s.path + ".compact.bak"
	_ = os.Remove(tempPath)
	_ = os.Remove(backupPath)

	tempDB, err := bolt.Open(tempPath, 0o600, &bolt.Options{Timeout: boltOpenTimeout})
	if err != nil {
		return err
	}

	compactErr := bolt.Compact(tempDB, s.db, 0)
	closeTempErr := tempDB.Close()
	if compactErr != nil {
		_ = os.Remove(tempPath)
		return compactErr
	}
	if closeTempErr != nil {
		_ = os.Remove(tempPath)
		return closeTempErr
	}

	if err := s.db.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	if err := os.Rename(s.path, backupPath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	restoreBackup := true
	defer func() {
		if restoreBackup {
			_ = os.Remove(s.path)
			_ = os.Rename(backupPath, s.path)
		}
		_ = os.Remove(tempPath)
	}()

	if err := os.Rename(tempPath, s.path); err != nil {
		return err
	}

	reopened, err := bolt.Open(s.path, 0o600, &bolt.Options{Timeout: boltOpenTimeout})
	if err != nil {
		return err
	}

	s.db = reopened
	restoreBackup = false
	_ = os.Remove(backupPath)
	return nil
}

func (s *BboltStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return s.db.Close()
}

type boltTx struct {
	tx       *bolt.Tx
	readOnly bool
}

func (tx *boltTx) EnsureNamespace(namespace string) error {
	if tx.readOnly {
		return ErrReadOnlyTx
	}

	if strings.TrimSpace(namespace) == "" {
		return ErrNamespaceRequired
	}

	_, err := tx.tx.CreateBucketIfNotExists([]byte(namespace))
	return err
}

func (tx *boltTx) Put(namespace, key string, value []byte) error {
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

	bucket := tx.tx.Bucket([]byte(namespace))
	if bucket == nil {
		return ErrNamespaceRequired
	}

	if err := bucket.Put([]byte(key), cloneBytes(value)); err != nil {
		return err
	}

	return nil
}

func (tx *boltTx) Get(namespace, key string) ([]byte, error) {
	if err := validateNamespaceKey(namespace, key); err != nil {
		return nil, err
	}

	bucket := tx.tx.Bucket([]byte(namespace))
	if bucket == nil {
		return nil, ErrNotFound
	}

	value := bucket.Get([]byte(key))
	if value == nil {
		return nil, ErrNotFound
	}

	return cloneBytes(value), nil
}

func (tx *boltTx) Delete(namespace, key string) error {
	if tx.readOnly {
		return ErrReadOnlyTx
	}

	if err := validateNamespaceKey(namespace, key); err != nil {
		return err
	}

	bucket := tx.tx.Bucket([]byte(namespace))
	if bucket == nil {
		return nil
	}

	return bucket.Delete([]byte(key))
}

func (tx *boltTx) List(namespace, prefix string, limit int) ([]Record, error) {
	if strings.TrimSpace(namespace) == "" {
		return nil, ErrNamespaceRequired
	}

	if limit < 0 {
		return nil, ErrNegativeListLimit
	}

	bucket := tx.tx.Bucket([]byte(namespace))
	if bucket == nil {
		return nil, nil
	}

	cursor := bucket.Cursor()
	records := make([]Record, 0)

	for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
		keyString := string(key)
		if prefix != "" && !strings.HasPrefix(keyString, prefix) {
			continue
		}

		if value == nil {
			continue
		}

		records = append(records, Record{
			Key:   keyString,
			Value: cloneBytes(value),
		})

		if limit > 0 && len(records) >= limit {
			break
		}
	}

	return records, nil
}

