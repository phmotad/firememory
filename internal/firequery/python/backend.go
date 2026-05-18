package python

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/firequery/models"
)

var (
	ErrPythonUnavailable  = errors.New("firequery/python: python backend is unavailable")
	ErrRunnerStartFailed  = errors.New("firequery/python: failed to start runner")
	ErrRunnerProtocol     = errors.New("firequery/python: invalid runner response")
	ErrRunnerHealthFailed = errors.New("firequery/python: runner healthcheck failed")
	ErrBackendClosed      = errors.New("firequery/python: backend is closed")
	ErrDimensionNotFound  = errors.New("firequery/python: model dimension not available")
)

//go:embed runner.py
var embeddedRunner string

type Options struct {
	PythonBin string
}

type Availability struct {
	Available bool
	Details   []string
}

type Backend struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	inPipe ioWriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	closed bool
}

type ioWriteCloser interface {
	Write([]byte) (int, error)
	Close() error
}

type request struct {
	Op      string   `json:"op"`
	ModelID string   `json:"model_id,omitempty"`
	Task    string   `json:"task,omitempty"`
	Text    string   `json:"text,omitempty"`
	Labels  []string `json:"labels,omitempty"`
	Texts   []string `json:"texts,omitempty"`
}

type response struct {
	OK        bool                 `json:"ok"`
	Error     string               `json:"error,omitempty"`
	Device    string               `json:"device,omitempty"`
	Dimension int                  `json:"dimension,omitempty"`
	Labels    []models.ScoredLabel `json:"labels,omitempty"`
	Entities  []models.Entity      `json:"entities,omitempty"`
	Vectors   [][]float32          `json:"vectors,omitempty"`
}

func DetectAvailability(ctx context.Context, options Options) Availability {
	pythonBin := defaultPythonBin(options.PythonBin)
	scriptPath, err := materializeRunner()
	if err != nil {
		return Availability{Available: false, Details: []string{err.Error()}}
	}

	cmd := exec.CommandContext(ctx, pythonBin, "-u", scriptPath, "--healthcheck")
	output, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return Availability{Available: false, Details: []string{detail}}
	}

	return Availability{
		Available: true,
		Details:   []string{"python model backend ready"},
	}
}

func Start(ctx context.Context, options Options) (*Backend, error) {
	pythonBin := defaultPythonBin(options.PythonBin)
	scriptPath, err := materializeRunner()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, pythonBin, "-u", scriptPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: stdin pipe: %v", ErrRunnerStartFailed, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("%w: stdout pipe: %v", ErrRunnerStartFailed, err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("%w: %v", ErrRunnerStartFailed, err)
	}

	backend := &Backend{
		cmd:    cmd,
		inPipe: stdin,
		stdin:  bufio.NewWriter(stdin),
		stdout: bufio.NewReader(stdout),
	}

	if err := backend.Health(ctx); err != nil {
		_ = backend.Close()
		return nil, err
	}

	return backend, nil
}

func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true
	_ = b.inPipe.Close()
	if b.cmd.Process != nil {
		_ = b.cmd.Process.Kill()
	}
	_, _ = b.cmd.Process.Wait()
	return nil
}

func (b *Backend) Health(ctx context.Context) error {
	var reply response
	if err := b.call(ctx, request{Op: "health"}, &reply); err != nil {
		return err
	}
	if !reply.OK {
		return fmt.Errorf("%w: %s", ErrRunnerHealthFailed, reply.Error)
	}
	return nil
}

func (b *Backend) ModelDimension(ctx context.Context, modelID, task string) (int, error) {
	var reply response
	if err := b.call(ctx, request{Op: "model_info", ModelID: modelID, Task: task}, &reply); err != nil {
		return 0, err
	}
	if !reply.OK {
		return 0, fmt.Errorf("%w: %s", ErrRunnerProtocol, reply.Error)
	}
	if reply.Dimension <= 0 && task != "entity_extraction" {
		return 0, ErrDimensionNotFound
	}
	return reply.Dimension, nil
}

func (b *Backend) Classify(ctx context.Context, modelID string, input models.TextInput, labels []string) ([]models.ScoredLabel, error) {
	var reply response
	if err := b.call(ctx, request{
		Op:      "classify",
		ModelID: modelID,
		Text:    input.Text,
		Labels:  labels,
	}, &reply); err != nil {
		return nil, err
	}
	if !reply.OK {
		return nil, fmt.Errorf("%w: %s", ErrRunnerProtocol, reply.Error)
	}
	return reply.Labels, nil
}

func (b *Backend) ExtractEntities(ctx context.Context, modelID string, input models.TextInput) ([]models.Entity, error) {
	var reply response
	if err := b.call(ctx, request{
		Op:      "extract_entities",
		ModelID: modelID,
		Text:    input.Text,
	}, &reply); err != nil {
		return nil, err
	}
	if !reply.OK {
		return nil, fmt.Errorf("%w: %s", ErrRunnerProtocol, reply.Error)
	}
	return reply.Entities, nil
}

func (b *Backend) EmbedTexts(ctx context.Context, modelID string, texts []string) ([]embedder.Vector, error) {
	var reply response
	if err := b.call(ctx, request{
		Op:      "embed",
		ModelID: modelID,
		Texts:   texts,
	}, &reply); err != nil {
		return nil, err
	}
	if !reply.OK {
		return nil, fmt.Errorf("%w: %s", ErrRunnerProtocol, reply.Error)
	}
	vectors := make([]embedder.Vector, 0, len(reply.Vectors))
	for _, vector := range reply.Vectors {
		copyVector := append(embedder.Vector(nil), vector...)
		vectors = append(vectors, copyVector)
	}
	return vectors, nil
}

func (b *Backend) call(_ context.Context, req request, reply *response) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrBackendClosed
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := b.stdin.Write(append(payload, '\n')); err != nil {
		return err
	}
	if err := b.stdin.Flush(); err != nil {
		return err
	}

	line, err := b.stdout.ReadBytes('\n')
	if err != nil {
		return err
	}
	if err := json.Unmarshal(line, reply); err != nil {
		return fmt.Errorf("%w: %v", ErrRunnerProtocol, err)
	}
	return nil
}

func materializeRunner() (string, error) {
	dir := filepath.Join(os.TempDir(), "firequery-python")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "firequery_model_runner.py")
	if err := os.WriteFile(path, []byte(embeddedRunner), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func defaultPythonBin(value string) string {
	if strings.TrimSpace(value) == "" {
		return "python"
	}
	return value
}

type TextClassificationClient struct {
	Backend *Backend
}

func (c TextClassificationClient) Classify(ctx context.Context, modelID string, input models.TextInput, labels []string) ([]models.ScoredLabel, error) {
	return c.Backend.Classify(ctx, modelID, input, labels)
}

type EntityExtractionClient struct {
	Backend *Backend
}

func (c EntityExtractionClient) ExtractEntities(ctx context.Context, modelID string, input models.TextInput) ([]models.Entity, error) {
	return c.Backend.ExtractEntities(ctx, modelID, input)
}

type QueryPassageEmbedder struct {
	Backend        *Backend
	ModelID        string
	DimensionValue int
}

func NewQueryPassageEmbedder(ctx context.Context, backend *Backend, modelID string) (*QueryPassageEmbedder, error) {
	dimension, err := backend.ModelDimension(ctx, modelID, "embedding")
	if err != nil {
		return nil, err
	}
	return &QueryPassageEmbedder{
		Backend:        backend,
		ModelID:        modelID,
		DimensionValue: dimension,
	}, nil
}

func (e *QueryPassageEmbedder) Name() string {
	return e.ModelID
}

func (e *QueryPassageEmbedder) Dimension() int {
	return e.DimensionValue
}

func (e *QueryPassageEmbedder) EmbedQuery(ctx context.Context, text string) (embedder.Vector, error) {
	vectors, err := e.Backend.EmbedTexts(ctx, e.ModelID, []string{"query: " + text})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

func (e *QueryPassageEmbedder) EmbedPassage(ctx context.Context, text string) (embedder.Vector, error) {
	vectors, err := e.Backend.EmbedTexts(ctx, e.ModelID, []string{"passage: " + text})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}
