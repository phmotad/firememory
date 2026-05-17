package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

const (
	DefaultPort        = 45555
	defaultIdleTimeout = 15 * time.Minute
	writeBufferSize    = 1000
)

// HandleFunc is the function signature the daemon calls to process a request.
// It is supplied by cmd/fquery/daemon.go and wraps the full ML pipeline.
type HandleFunc func(context.Context, contract.ExternalRequest) (contract.ExternalResponse, error)

// Config holds everything the daemon needs to start.
type Config struct {
	HandleFn    HandleFunc
	ErrLog      io.Writer
	Port        int
	IdleTimeout time.Duration
	// ShutdownFn is called just before os.Exit(0) on idle timeout.
	ShutdownFn func()
}

// Server is the ephemeral local daemon. It owns the brain connection via
// HandleFn, serialises writes through a buffered channel, and exits after
// being idle for IdleTimeout.
type Server struct {
	handleFn   HandleFunc
	writeCh    chan writeJob
	activity   *activityMonitor
	errLog     io.Writer
	httpServer *http.Server
	shutdownFn func()
	idleTO     time.Duration
}

func New(cfg Config) *Server {
	if cfg.Port == 0 {
		cfg.Port = DefaultPort
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}
	if cfg.ShutdownFn == nil {
		cfg.ShutdownFn = func() {}
	}
	if cfg.ErrLog == nil {
		cfg.ErrLog = io.Discard
	}

	s := &Server{
		handleFn:   cfg.HandleFn,
		writeCh:    make(chan writeJob, writeBufferSize),
		activity:   &activityMonitor{},
		errLog:     cfg.ErrLog,
		shutdownFn: cfg.ShutdownFn,
		idleTO:     cfg.IdleTimeout,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", s.handlePing)
	mux.HandleFunc("POST /v1/request", s.handleRequest)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.Port),
		Handler: mux,
	}

	return s
}

// Start begins serving requests. It blocks until the server is closed.
func (s *Server) Start() error {
	go startWriteWorker(s.writeCh, s.handleFn, s.activity, s.errLog)
	go s.activity.watchAndExit(s.idleTO, func() {
		_ = s.httpServer.Shutdown(context.Background())
		s.shutdownFn()
	})

	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("daemon: listen %s: %w", s.httpServer.Addr, err)
	}

	fmt.Fprintf(s.errLog, "daemon: listening on %s (idle timeout: %s)\n",
		s.httpServer.Addr, s.idleTO)

	if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handlePing(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	var req contract.ExternalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"ok":false,"error":{"code":"BAD_REQUEST","message":"invalid JSON body"}}`, http.StatusBadRequest)
		return
	}

	s.activity.touch()

	if isWriteOperation(req.Operation) {
		select {
		case s.writeCh <- writeJob{ctx: r.Context(), request: req}:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			writeJSON(w, contract.ExternalResponse{
				OK:        true,
				RequestID: req.RequestID,
				Operation: req.Operation,
				Data:      map[string]any{"status": "queued"},
			})
		default:
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"ok":false,"error":{"code":"QUEUE_FULL","message":"write queue full, try again"}}`, http.StatusServiceUnavailable)
		}
		return
	}

	// Synchronous read path.
	resp, err := s.handleFn(r.Context(), req)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, contract.ExternalResponse{
			OK:        false,
			RequestID: req.RequestID,
			Operation: req.Operation,
			Error: &contract.Error{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	writeJSON(w, resp)
}

// isWriteOperation returns true for operations that mutate the brain.
func isWriteOperation(operation string) bool {
	switch operation {
	case "remember", "sync":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}
