package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"

	"github.com/phmotad/firememory/internal/graph"
	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/storage"
)

// Server serves the knowledge graph UI on a local HTTP port.
type Server struct {
	store      storage.Store
	httpServer *http.Server
	addr       string
}

// NodeDetail is the response for /api/node/{id}.
type NodeDetail struct {
	ID         string            `json:"id"`
	Label      string            `json:"label"`
	Kind       memory.MemoryKind `json:"kind"`
	Scope      string            `json:"scope"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Content    string            `json:"content,omitempty"`
	Importance float64           `json:"importance,omitempty"`
	Confidence float64           `json:"confidence,omitempty"`
	CreatedAt  string            `json:"created_at"`
	UpdatedAt  string            `json:"updated_at"`
	Edges      []EdgeSummary     `json:"edges"`
}

// EdgeSummary is the compact edge representation used inside NodeDetail.
type EdgeSummary struct {
	ID     string              `json:"id"`
	FromID string              `json:"from_id"`
	ToID   string              `json:"to_id"`
	Type   memory.RelationType `json:"type"`
	Weight float64             `json:"weight"`
}

// GraphResponse is the response for /api/graph.
type GraphResponse struct {
	Nodes []graph.Node `json:"nodes"`
	Edges []graph.Edge `json:"edges"`
}

// New creates a server that reads from store and listens on the given port.
func New(store storage.Store, port int) *Server {
	s := &Server{
		store: store,
		addr:  fmt.Sprintf("127.0.0.1:%d", port),
	}

	assets, _ := fs.Sub(Assets, "assets")
	staticHandler := http.FileServer(http.FS(assets))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/graph", s.handleGraph)
	mux.HandleFunc("GET /api/node/", s.handleNode)
	mux.Handle("/", staticHandler)

	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	return s
}

// Start listens and serves until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("ui: listen %s: %w", s.addr, err)
	}

	go func() {
		<-ctx.Done()
		_ = s.httpServer.Shutdown(context.Background())
	}()

	if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handleGraph(w http.ResponseWriter, _ *http.Request) {
	nodes, edges, err := s.loadAll()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, GraphResponse{Nodes: nodes, Edges: edges})
}

func (s *Server) handleNode(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/node/")
	if id == "" {
		http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
		return
	}

	raw, err := s.store.Get(graph.NodesNamespace, id)
	if err != nil {
		if err == storage.ErrNotFound {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		writeError(w, err)
		return
	}

	var node graph.Node
	if err := json.Unmarshal(raw, &node); err != nil {
		writeError(w, err)
		return
	}

	detail := NodeDetail{
		ID:        node.ID,
		Label:     node.Label,
		Kind:      node.Kind,
		Scope:     node.Scope,
		Metadata:  node.Metadata,
		CreatedAt: node.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: node.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Edges:     []EdgeSummary{},
	}

	// Enrich with Memory content/scores if the node ID maps to a stored memory.
	if memRaw, err := s.store.Get("memories", id); err == nil {
		var m memory.Memory
		if json.Unmarshal(memRaw, &m) == nil {
			detail.Content = m.Content
			detail.Importance = m.Importance
			detail.Confidence = m.Confidence
		}
	}

	// Collect edges connected to this node.
	edgeRecords, err := s.store.List(graph.EdgesNamespace, "", 0)
	if err == nil {
		for _, rec := range edgeRecords {
			var e graph.Edge
			if json.Unmarshal(rec.Value, &e) != nil {
				continue
			}
			if e.FromID == id || e.ToID == id {
				detail.Edges = append(detail.Edges, EdgeSummary{
					ID:     e.ID,
					FromID: e.FromID,
					ToID:   e.ToID,
					Type:   e.Type,
					Weight: e.Weight,
				})
			}
		}
	}

	writeJSON(w, detail)
}

func (s *Server) loadAll() ([]graph.Node, []graph.Edge, error) {
	nodeRecords, err := s.store.List(graph.NodesNamespace, "", 0)
	if err != nil {
		return nil, nil, err
	}
	nodes := make([]graph.Node, 0, len(nodeRecords))
	for _, rec := range nodeRecords {
		var n graph.Node
		if json.Unmarshal(rec.Value, &n) == nil {
			nodes = append(nodes, n)
		}
	}

	edgeRecords, err := s.store.List(graph.EdgesNamespace, "", 0)
	if err != nil {
		return nil, nil, err
	}
	edges := make([]graph.Edge, 0, len(edgeRecords))
	for _, rec := range edgeRecords {
		var e graph.Edge
		if json.Unmarshal(rec.Value, &e) == nil {
			edges = append(edges, e)
		}
	}

	return nodes, edges, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
