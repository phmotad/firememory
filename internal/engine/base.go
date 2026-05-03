package engine

import (
	"encoding/json"
	"errors"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/dedup"
	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/extractor"
	"github.com/phmotad/firememory/internal/graph"
	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/storage"
	"github.com/phmotad/firememory/internal/vector"
)

const (
	memoriesNamespace  = "memories"
	vectorsNamespace   = "vectors"
	hashIndexNamespace = "hash_index"
	syncQueueNamespace = "sync_queue"
	tracesNamespace    = "traces"
)

var (
	ErrUnsupportedEmbeddingModel    = errors.New("unsupported embedding model")
	ErrEmbedderDimensionMismatch    = errors.New("embedder dimension does not match brain manifest")
	ErrEmbedderNameMismatch         = errors.New("embedder name does not match brain manifest")
	ErrVectorIndexDimensionMismatch = errors.New("vector index dimension does not match brain manifest")
	ErrBrainPathMismatch            = errors.New("brain path does not match the opened engine")
)

type Engine interface {
	Path() string
	Manifest() brainfile.Manifest
	Store() storage.Store
	Embedder() embedder.Embedder
	VectorIndex() vector.VectorIndex
	Graph() graph.Graph
	Remember(input RememberInput) (RememberResult, error)
	Recall(input RecallInput) (RecallResult, error)
	Context(input ContextInput) (ContextResult, error)
	Sync(input SyncInput) (SyncResult, error)
	Explain(input ExplainInput) (ExplainResult, error)
	Close() error
}

type Options struct {
	Path               string
	Embedder           embedder.Embedder
	VectorIndex        vector.VectorIndex
	Graph              graph.Graph
	Extractor          extractor.Extractor
	RelationClassifier MemoryRelationClassifier
}

type Base struct {
	brainfile          *brainfile.Handle
	embedder           embedder.Embedder
	vectorIndex        vector.VectorIndex
	graph              graph.Graph
	hashIndex          *dedup.InMemoryHashIndex
	extractor          extractor.Extractor
	relationClassifier MemoryRelationClassifier
}

type storedVectorRecord struct {
	Vector embedder.Vector   `json:"vector"`
	Scope  string            `json:"scope"`
	Kind   memory.MemoryKind `json:"kind"`
}

func Open(opts Options) (*Base, error) {
	if err := validateBrainPath(opts.Path); err != nil {
		return nil, err
	}

	handle, err := brainfile.Open(opts.Path)
	if err != nil {
		return nil, err
	}

	manifest := handle.Manifest()

	resolvedEmbedder, err := resolveEmbedder(manifest, opts.Embedder)
	if err != nil {
		_ = handle.Close()
		return nil, err
	}

	resolvedIndex, err := resolveVectorIndex(manifest, opts.VectorIndex)
	if err != nil {
		_ = handle.Close()
		return nil, err
	}

	if err := rebuildVectorIndex(handle.Store(), resolvedIndex); err != nil {
		_ = handle.Close()
		return nil, err
	}

	hashIndex, err := rebuildHashIndex(handle.Store())
	if err != nil {
		_ = handle.Close()
		return nil, err
	}

	resolvedGraph, err := resolveGraph(handle.Store(), opts.Graph)
	if err != nil {
		_ = handle.Close()
		return nil, err
	}

	resolvedExtractor := resolveExtractor(opts.Extractor)
	resolvedRelationClassifier := resolveRelationClassifier(opts.RelationClassifier)

	return &Base{
		brainfile:          handle,
		embedder:           resolvedEmbedder,
		vectorIndex:        resolvedIndex,
		graph:              resolvedGraph,
		hashIndex:          hashIndex,
		extractor:          resolvedExtractor,
		relationClassifier: resolvedRelationClassifier,
	}, nil
}

func (e *Base) Path() string {
	return e.brainfile.Path()
}

func (e *Base) Manifest() brainfile.Manifest {
	return e.brainfile.Manifest()
}

func (e *Base) Store() storage.Store {
	return e.brainfile.Store()
}

func (e *Base) Embedder() embedder.Embedder {
	return e.embedder
}

func (e *Base) VectorIndex() vector.VectorIndex {
	return e.vectorIndex
}

func (e *Base) Graph() graph.Graph {
	return e.graph
}

func (e *Base) Close() error {
	return e.brainfile.Close()
}

func resolveEmbedder(manifest brainfile.Manifest, injected embedder.Embedder) (embedder.Embedder, error) {
	if injected != nil {
		if injected.Dimension() != manifest.EmbeddingDim {
			return nil, ErrEmbedderDimensionMismatch
		}

		if manifest.EmbeddingModel != "" && injected.Name() != manifest.EmbeddingModel {
			return nil, ErrEmbedderNameMismatch
		}

		return injected, nil
	}

	switch manifest.EmbeddingModel {
	case "", brainfile.DefaultEmbedder:
		return embedder.NewDeterministicEmbedder(manifest.EmbeddingDim)
	default:
		return nil, ErrUnsupportedEmbeddingModel
	}
}

func resolveVectorIndex(manifest brainfile.Manifest, injected vector.VectorIndex) (vector.VectorIndex, error) {
	if injected != nil {
		if injected.Dimension() != manifest.EmbeddingDim {
			return nil, ErrVectorIndexDimensionMismatch
		}

		return injected, nil
	}

	return vector.NewLinearVectorIndex(manifest.EmbeddingDim)
}

func resolveGraph(store storage.Store, injected graph.Graph) (graph.Graph, error) {
	if injected != nil {
		return injected, nil
	}

	return graph.New(store)
}

func resolveExtractor(injected extractor.Extractor) extractor.Extractor {
	if injected != nil {
		return injected
	}

	return extractor.NewHeuristicExtractor()
}

func resolveRelationClassifier(injected MemoryRelationClassifier) MemoryRelationClassifier {
	if injected != nil {
		return injected
	}

	return NewHeuristicMemoryRelationClassifier()
}

func rebuildVectorIndex(store storage.Store, idx vector.VectorIndex) error {
	records, err := store.List(vectorsNamespace, "", 0)
	if err != nil {
		return err
	}

	for _, record := range records {
		var stored storedVectorRecord
		if err := json.Unmarshal(record.Value, &stored); err != nil {
			return err
		}

		entry := vector.Entry{
			ID:     record.Key,
			Vector: stored.Vector,
			Scope:  stored.Scope,
			Kind:   stored.Kind,
		}

		if err := idx.Add(entry); err != nil {
			return err
		}
	}

	return nil
}

func rebuildHashIndex(store storage.Store) (*dedup.InMemoryHashIndex, error) {
	index := dedup.NewInMemoryHashIndex()

	records, err := store.List(hashIndexNamespace, "", 0)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		index.Set(record.Key, string(record.Value))
	}

	return index, nil
}
