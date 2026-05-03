package graph

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/storage"
)

const (
	NodesNamespace = "graph_nodes"
	EdgesNamespace = "graph_edges"
)

var (
	ErrNodeIDRequired    = errors.New("node id is required")
	ErrEdgeIDRequired    = errors.New("edge id is required")
	ErrNodeNotFound      = errors.New("node not found")
	ErrDepthNegative     = errors.New("depth cannot be negative")
	ErrInvalidEdgeType   = errors.New("invalid edge type")
	ErrInvalidEdgeWeight = errors.New("edge weight must be between 0 and 1")
)

type Node struct {
	ID        string            `json:"id"`
	Label     string            `json:"label"`
	Kind      memory.MemoryKind `json:"kind"`
	Scope     string            `json:"scope"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func (n *Node) Normalize() {
	if n.Kind == "" {
		n.Kind = memory.MemoryKindNote
	}

	if strings.TrimSpace(n.Scope) == "" {
		n.Scope = memory.DefaultScope
	}

	if n.Metadata == nil {
		n.Metadata = map[string]string{}
	}

	now := time.Now().UTC()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = now
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}
}

func (n Node) Validate() error {
	if strings.TrimSpace(n.ID) == "" {
		return ErrNodeIDRequired
	}

	if !n.Kind.Valid() {
		return memory.ErrInvalidMemoryKind
	}

	return nil
}

type Edge struct {
	ID        string              `json:"id"`
	FromID    string              `json:"from_id"`
	ToID      string              `json:"to_id"`
	Type      memory.RelationType `json:"type"`
	Weight    float64             `json:"weight"`
	CreatedAt time.Time           `json:"created_at"`
}

func (e *Edge) Normalize() {
	if e.Weight == 0 {
		e.Weight = 1
	}

	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
}

func (e Edge) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return ErrEdgeIDRequired
	}

	if strings.TrimSpace(e.FromID) == "" || strings.TrimSpace(e.ToID) == "" {
		return memory.ErrEmptyRelationSide
	}

	if !e.Type.Valid() {
		return ErrInvalidEdgeType
	}

	if e.Weight < 0 || e.Weight > 1 {
		return ErrInvalidEdgeWeight
	}

	return nil
}

type Graph interface {
	AddNode(node Node) error
	AddEdge(edge Edge) error
	GetNode(id string) (Node, error)
	Neighbors(id string) ([]Node, error)
	Related(id string, depth int) ([]Node, error)
	TraverseDepth(id string, depth int) ([]Node, error)
	NodeCount() int
	EdgeCount() int
}

type PersistentGraph struct {
	mu        sync.RWMutex
	store     storage.Store
	nodes     map[string]Node
	edges     map[string]Edge
	adjacency map[string]map[string]struct{}
}

func New(store storage.Store) (*PersistentGraph, error) {
	g := &PersistentGraph{
		store:     store,
		nodes:     map[string]Node{},
		edges:     map[string]Edge{},
		adjacency: map[string]map[string]struct{}{},
	}

	if store == nil {
		return g, nil
	}

	if err := store.EnsureNamespace(NodesNamespace); err != nil {
		return nil, err
	}

	if err := store.EnsureNamespace(EdgesNamespace); err != nil {
		return nil, err
	}

	if err := g.load(); err != nil {
		return nil, err
	}

	return g, nil
}

func (g *PersistentGraph) AddNode(node Node) error {
	node.Normalize()
	if err := node.Validate(); err != nil {
		return err
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if existing, ok := g.nodes[node.ID]; ok {
		node.CreatedAt = existing.CreatedAt
		if node.UpdatedAt.Before(existing.UpdatedAt) {
			node.UpdatedAt = time.Now().UTC()
		}
	}

	if err := g.persistNode(node); err != nil {
		return err
	}

	g.nodes[node.ID] = cloneNode(node)
	if g.adjacency[node.ID] == nil {
		g.adjacency[node.ID] = map[string]struct{}{}
	}

	return nil
}

func (g *PersistentGraph) AddEdge(edge Edge) error {
	edge.Normalize()
	if err := edge.Validate(); err != nil {
		return err
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.nodes[edge.FromID]; !ok {
		return ErrNodeNotFound
	}

	if _, ok := g.nodes[edge.ToID]; !ok {
		return ErrNodeNotFound
	}

	if err := g.persistEdge(edge); err != nil {
		return err
	}

	g.edges[edge.ID] = edge
	g.linkEdge(edge)
	return nil
}

func (g *PersistentGraph) GetNode(id string) (Node, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, ok := g.nodes[id]
	if !ok {
		return Node{}, ErrNodeNotFound
	}

	return cloneNode(node), nil
}

func (g *PersistentGraph) Neighbors(id string) ([]Node, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[id]; !ok {
		return nil, ErrNodeNotFound
	}

	return g.neighborsLocked(id), nil
}

func (g *PersistentGraph) Related(id string, depth int) ([]Node, error) {
	return g.TraverseDepth(id, depth)
}

func (g *PersistentGraph) TraverseDepth(id string, depth int) ([]Node, error) {
	if depth < 0 {
		return nil, ErrDepthNegative
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[id]; !ok {
		return nil, ErrNodeNotFound
	}

	if depth == 0 {
		return nil, nil
	}

	visited := map[string]struct{}{
		id: {},
	}
	results := make([]Node, 0)

	var dfs func(current string, remaining int)
	dfs = func(current string, remaining int) {
		if remaining == 0 {
			return
		}

		neighborIDs := g.neighborIDsLocked(current)
		for _, neighborID := range neighborIDs {
			if _, ok := visited[neighborID]; ok {
				continue
			}

			visited[neighborID] = struct{}{}
			results = append(results, cloneNode(g.nodes[neighborID]))
			dfs(neighborID, remaining-1)
		}
	}

	dfs(id, depth)
	return results, nil
}

func (g *PersistentGraph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

func (g *PersistentGraph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.edges)
}

func (g *PersistentGraph) load() error {
	nodeRecords, err := g.store.List(NodesNamespace, "", 0)
	if err != nil {
		return err
	}

	for _, record := range nodeRecords {
		var node Node
		if err := json.Unmarshal(record.Value, &node); err != nil {
			return err
		}

		node.Normalize()
		if err := node.Validate(); err != nil {
			return err
		}

		g.nodes[node.ID] = cloneNode(node)
		if g.adjacency[node.ID] == nil {
			g.adjacency[node.ID] = map[string]struct{}{}
		}
	}

	edgeRecords, err := g.store.List(EdgesNamespace, "", 0)
	if err != nil {
		return err
	}

	for _, record := range edgeRecords {
		var edge Edge
		if err := json.Unmarshal(record.Value, &edge); err != nil {
			return err
		}

		edge.Normalize()
		if err := edge.Validate(); err != nil {
			return err
		}

		g.edges[edge.ID] = edge
		g.linkEdge(edge)
	}

	return nil
}

func (g *PersistentGraph) persistNode(node Node) error {
	if g.store == nil {
		return nil
	}

	payload, err := json.Marshal(node)
	if err != nil {
		return err
	}

	return g.store.Put(NodesNamespace, node.ID, payload)
}

func (g *PersistentGraph) persistEdge(edge Edge) error {
	if g.store == nil {
		return nil
	}

	payload, err := json.Marshal(edge)
	if err != nil {
		return err
	}

	return g.store.Put(EdgesNamespace, edge.ID, payload)
}

func (g *PersistentGraph) linkEdge(edge Edge) {
	if g.adjacency[edge.FromID] == nil {
		g.adjacency[edge.FromID] = map[string]struct{}{}
	}
	if g.adjacency[edge.ToID] == nil {
		g.adjacency[edge.ToID] = map[string]struct{}{}
	}

	g.adjacency[edge.FromID][edge.ToID] = struct{}{}
	g.adjacency[edge.ToID][edge.FromID] = struct{}{}
}

func (g *PersistentGraph) neighborsLocked(id string) []Node {
	neighborIDs := g.neighborIDsLocked(id)
	nodes := make([]Node, 0, len(neighborIDs))
	for _, neighborID := range neighborIDs {
		nodes = append(nodes, cloneNode(g.nodes[neighborID]))
	}
	return nodes
}

func (g *PersistentGraph) neighborIDsLocked(id string) []string {
	adjacent := g.adjacency[id]
	ids := make([]string, 0, len(adjacent))
	for neighborID := range adjacent {
		ids = append(ids, neighborID)
	}
	sort.Strings(ids)
	return ids
}

func cloneNode(node Node) Node {
	cloned := node
	if node.Metadata != nil {
		cloned.Metadata = make(map[string]string, len(node.Metadata))
		for key, value := range node.Metadata {
			cloned.Metadata[key] = value
		}
	}
	return cloned
}
