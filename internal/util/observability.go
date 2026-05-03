package util

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/engine"
	"github.com/phmotad/firememory/internal/extractor"
	"github.com/phmotad/firememory/internal/graph"
	"github.com/phmotad/firememory/internal/storage"
)

type DiagnosticError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Trace struct {
	Component string   `json:"component"`
	Steps     []string `json:"steps"`
}

type LogEvent struct {
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"`
	Component string         `json:"component"`
	Operation string         `json:"operation,omitempty"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type JSONLogger struct {
	Writer io.Writer
}

func NewJSONLogger(writer io.Writer) JSONLogger {
	return JSONLogger{Writer: writer}
}

func (l JSONLogger) Log(event LogEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if strings.TrimSpace(event.Level) == "" {
		event.Level = "info"
	}
	return WriteJSONLine(l.Writer, event)
}

func ErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, storage.ErrStoreLocked):
		return "FMEM_STORAGE_LOCKED"
	case errors.Is(err, storage.ErrStoreClosed):
		return "FMEM_STORAGE_CLOSED"
	case errors.Is(err, storage.ErrNotFound):
		return "FMEM_NOT_FOUND"
	case errors.Is(err, storage.ErrPathRequired):
		return "FMEM_PATH_REQUIRED"
	case errors.Is(err, brainfile.ErrInvalidExtension):
		return "FMEM_INVALID_EXTENSION"
	case errors.Is(err, brainfile.ErrManifestNotFound):
		return "FMEM_MANIFEST_NOT_FOUND"
	case errors.Is(err, brainfile.ErrIntegrityViolation):
		return "FMEM_INTEGRITY_VIOLATION"
	case errors.Is(err, brainfile.ErrUnsupportedFormatVersion):
		return "FMEM_UNSUPPORTED_FORMAT_VERSION"
	case errors.Is(err, engine.ErrBrainPathRequired):
		return "FMEM_BRAIN_PATH_REQUIRED"
	case errors.Is(err, engine.ErrBrainPathExtension):
		return "FMEM_BRAIN_PATH_EXTENSION"
	case errors.Is(err, engine.ErrQueryRequired):
		return "FMEM_QUERY_REQUIRED"
	case errors.Is(err, engine.ErrContentRequired):
		return "FMEM_CONTENT_REQUIRED"
	case errors.Is(err, engine.ErrInvalidTopK):
		return "FMEM_INVALID_TOP_K"
	case errors.Is(err, engine.ErrInvalidBudgetTokens):
		return "FMEM_INVALID_BUDGET_TOKENS"
	case errors.Is(err, engine.ErrConfirmationRequired):
		return "FMEM_CONFIRMATION_REQUIRED"
	case errors.Is(err, embedder.ErrDimensionMismatch):
		return "FMEM_EMBEDDING_DIMENSION_MISMATCH"
	case errors.Is(err, extractor.ErrGLiNERUnavailable):
		return "FMEM_EXTRACTOR_UNAVAILABLE"
	case errors.Is(err, graph.ErrNodeNotFound):
		return "FMEM_GRAPH_NODE_NOT_FOUND"
	case strings.Contains(err.Error(), "firequery: mandatory specialists are not healthy"):
		return "FQUERY_NOT_READY"
	case strings.Contains(err.Error(), "firequery: pipeline is required"):
		return "FQUERY_PIPELINE_REQUIRED"
	case strings.Contains(err.Error(), "firequery: runtime is required"):
		return "FQUERY_RUNTIME_REQUIRED"
	default:
		return "INTERNAL_ERROR"
	}
}

func ErrorDiagnostic(err error) DiagnosticError {
	return DiagnosticError{
		Code:    ErrorCode(err),
		Message: err.Error(),
	}
}

func StructuredTrace(component string, steps []string) Trace {
	return Trace{
		Component: component,
		Steps:     append([]string(nil), steps...),
	}
}

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func WriteJSONLine(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(value)
}

func ExtractJSONFlag(args []string) ([]string, bool) {
	filtered := make([]string, 0, len(args))
	jsonOutput := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, jsonOutput
}

func IsTruthyEnv(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}
