package storage

import (
	"errors"
	"time"
)

var (
	ErrNamespaceRequired = errors.New("namespace is required")
	ErrKeyRequired       = errors.New("key is required")
	ErrValueRequired     = errors.New("value is required")
	ErrPathRequired      = errors.New("path is required")
	ErrStoreLocked       = errors.New("store is locked by another process")
	ErrStoreClosed       = errors.New("store is closed")
	ErrReadOnlyTx        = errors.New("transaction is read-only")
	ErrNotFound          = errors.New("record not found")
	ErrNegativeListLimit = errors.New("list limit cannot be negative")
)

type Record struct {
	Key   string
	Value []byte
}

type Snapshot struct {
	TakenAt    time.Time
	Namespaces map[string][]Record
}

type Tx interface {
	EnsureNamespace(namespace string) error
	Put(namespace, key string, value []byte) error
	Get(namespace, key string) ([]byte, error)
	Delete(namespace, key string) error
	List(namespace, prefix string, limit int) ([]Record, error)
}

type Store interface {
	EnsureNamespace(namespace string) error
	Put(namespace, key string, value []byte) error
	Get(namespace, key string) ([]byte, error)
	Delete(namespace, key string) error
	List(namespace, prefix string, limit int) ([]Record, error)
	View(fn func(Tx) error) error
	Update(fn func(Tx) error) error
	Snapshot() (Snapshot, error)
	Compact() error
	Close() error
}
