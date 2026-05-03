package runtime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type DeviceKind string

const (
	DeviceCPU      DeviceKind = "cpu"
	DeviceCUDA     DeviceKind = "cuda"
	DeviceDirectML DeviceKind = "directml"
	DeviceCoreML   DeviceKind = "coreml"
	DeviceOpenVINO DeviceKind = "openvino"
)

type Backend string

const (
	BackendCPU      Backend = "cpu"
	BackendCUDA     Backend = "cuda"
	BackendDirectML Backend = "directml"
	BackendCoreML   Backend = "coreml"
	BackendOpenVINO Backend = "openvino"
)

var (
	ErrNoAvailableBackend     = errors.New("firequery/runtime: no available backend")
	ErrGPUNotAvailable        = errors.New("firequery/runtime: gpu backend not available")
	ErrModelNotRegistered     = errors.New("firequery/runtime: model is not registered")
	ErrMemoryBudgetExceeded   = errors.New("firequery/runtime: memory budget exceeded")
	ErrLoaderRequired         = errors.New("firequery/runtime: loader is required")
	ErrModelIDRequired        = errors.New("firequery/runtime: model id is required")
	ErrModelVersionRequired   = errors.New("firequery/runtime: model version is required")
	ErrNoSupportedBackends    = errors.New("firequery/runtime: model must support at least one backend")
	ErrSelectorRequired       = errors.New("firequery/runtime: selector is required")
	ErrDetectorRequired       = errors.New("firequery/runtime: detector is required")
	ErrRegistryRequired       = errors.New("firequery/runtime: registry is required")
)

type Device struct {
	Kind        DeviceKind
	Name        string
	MemoryBytes uint64
	Available   bool
}

func (d Device) Supports() Backend {
	switch d.Kind {
	case DeviceCUDA:
		return BackendCUDA
	case DeviceDirectML:
		return BackendDirectML
	case DeviceCoreML:
		return BackendCoreML
	case DeviceOpenVINO:
		return BackendOpenVINO
	default:
		return BackendCPU
	}
}

type SelectionRequest struct {
	ModelID     string
	Preferred   Backend
	RequireGPU  bool
	MemoryBytes uint64
	Supported   []Backend
}

type Health struct {
	Ready   bool
	Backend Backend
	Notes   []string
	Devices []Device
	Models  []ModelState
}

type ModelSpec struct {
	ID          string
	Version     string
	Backends    []Backend
	MemoryBytes uint64
	Loader      Loader
}

type ModelState struct {
	ID          string
	Version     string
	Backend     Backend
	Loaded      bool
	Healthy     bool
	MemoryBytes uint64
	Notes       []string
}

type Loader func(ctx context.Context, backend Backend) error

type DeviceDetector interface {
	Detect(ctx context.Context) ([]Device, error)
}

type BackendSelector interface {
	Select(ctx context.Context, devices []Device, request SelectionRequest) (Backend, error)
}

type Manager interface {
	Devices(ctx context.Context) ([]Device, error)
	Health(ctx context.Context) (Health, error)
	RegisterModel(spec ModelSpec) error
	EnsureLoaded(ctx context.Context, modelID string, request SelectionRequest) (Backend, error)
	States() []ModelState
}

type envLookupFunc func(string) string

type EnvironmentDetector struct {
	LookupEnv envLookupFunc
}

func NewEnvironmentDetector() EnvironmentDetector {
	return EnvironmentDetector{}
}

func (d EnvironmentDetector) Detect(context.Context) ([]Device, error) {
	lookup := d.LookupEnv
	if lookup == nil {
		lookup = func(string) string { return "" }
	}

	devices := []Device{
		{Kind: DeviceCPU, Name: "cpu", Available: true},
	}

	type optionalDevice struct {
		env  string
		kind DeviceKind
		name string
	}

	options := []optionalDevice{
		{env: "FIREQUERY_ENABLE_CUDA", kind: DeviceCUDA, name: "cuda"},
		{env: "FIREQUERY_ENABLE_DIRECTML", kind: DeviceDirectML, name: "directml"},
		{env: "FIREQUERY_ENABLE_COREML", kind: DeviceCoreML, name: "coreml"},
		{env: "FIREQUERY_ENABLE_OPENVINO", kind: DeviceOpenVINO, name: "openvino"},
	}

	for _, option := range options {
		if isTruthy(lookup(option.env)) {
			devices = append(devices, Device{
				Kind:      option.kind,
				Name:      option.name,
				Available: true,
			})
		}
	}

	return devices, nil
}

type LocalBackendSelector struct{}

func (LocalBackendSelector) Select(_ context.Context, devices []Device, request SelectionRequest) (Backend, error) {
	available := availableBackends(devices)
	supported := normalizeSupported(request.Supported)

	if request.Preferred != "" && containsBackend(available, request.Preferred) && containsBackend(supported, request.Preferred) {
		return request.Preferred, nil
	}

	if request.RequireGPU {
		for _, backend := range []Backend{BackendCUDA, BackendDirectML, BackendCoreML, BackendOpenVINO} {
			if containsBackend(available, backend) && containsBackend(supported, backend) {
				return backend, nil
			}
		}
		return "", ErrGPUNotAvailable
	}

	for _, backend := range []Backend{BackendCUDA, BackendDirectML, BackendCoreML, BackendOpenVINO, BackendCPU} {
		if containsBackend(available, backend) && containsBackend(supported, backend) {
			return backend, nil
		}
	}

	return "", ErrNoAvailableBackend
}

type Registry struct {
	mu             sync.Mutex
	maxMemoryBytes uint64
	usedMemory     uint64
	specs          map[string]ModelSpec
	states         map[string]ModelState
}

func NewRegistry(maxMemoryBytes uint64) *Registry {
	return &Registry{
		maxMemoryBytes: maxMemoryBytes,
		specs:          make(map[string]ModelSpec),
		states:         make(map[string]ModelState),
	}
}

func (r *Registry) Register(spec ModelSpec) error {
	if strings.TrimSpace(spec.ID) == "" {
		return ErrModelIDRequired
	}
	if strings.TrimSpace(spec.Version) == "" {
		return ErrModelVersionRequired
	}
	if len(spec.Backends) == 0 {
		return ErrNoSupportedBackends
	}
	if spec.Loader == nil {
		return ErrLoaderRequired
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.specs[spec.ID] = spec
	if _, ok := r.states[spec.ID]; !ok {
		r.states[spec.ID] = ModelState{
			ID:          spec.ID,
			Version:     spec.Version,
			MemoryBytes: spec.MemoryBytes,
			Loaded:      false,
			Healthy:     true,
			Notes:       []string{"registered"},
		}
	}
	return nil
}

func (r *Registry) EnsureLoaded(ctx context.Context, modelID string, backend Backend) error {
	r.mu.Lock()
	spec, ok := r.specs[modelID]
	state := r.states[modelID]
	if !ok {
		r.mu.Unlock()
		return ErrModelNotRegistered
	}
	if state.Loaded {
		r.mu.Unlock()
		return nil
	}
	if !containsBackend(spec.Backends, backend) {
		r.mu.Unlock()
		return fmt.Errorf("%w: backend %s is not supported by model %s", ErrNoAvailableBackend, backend, modelID)
	}
	if r.maxMemoryBytes > 0 && r.usedMemory+spec.MemoryBytes > r.maxMemoryBytes {
		state.Healthy = false
		state.Notes = []string{"memory budget exceeded"}
		r.states[modelID] = state
		r.mu.Unlock()
		return ErrMemoryBudgetExceeded
	}
	r.mu.Unlock()

	if err := spec.Loader(ctx, backend); err != nil {
		r.mu.Lock()
		state = r.states[modelID]
		state.Healthy = false
		state.Backend = backend
		state.Notes = []string{err.Error()}
		r.states[modelID] = state
		r.mu.Unlock()
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	state = r.states[modelID]
	state.Loaded = true
	state.Healthy = true
	state.Backend = backend
	state.Notes = []string{"loaded lazily"}
	r.states[modelID] = state
	r.usedMemory += spec.MemoryBytes
	return nil
}

func (r *Registry) States() []ModelState {
	r.mu.Lock()
	defer r.mu.Unlock()

	states := make([]ModelState, 0, len(r.states))
	for _, state := range r.states {
		state.Notes = append([]string(nil), state.Notes...)
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].ID < states[j].ID
	})
	return states
}

type LocalManager struct {
	detector DeviceDetector
	selector BackendSelector
	registry *Registry
}

func NewManager(detector DeviceDetector, selector BackendSelector, registry *Registry) (*LocalManager, error) {
	if detector == nil {
		return nil, ErrDetectorRequired
	}
	if selector == nil {
		return nil, ErrSelectorRequired
	}
	if registry == nil {
		return nil, ErrRegistryRequired
	}
	return &LocalManager{
		detector: detector,
		selector: selector,
		registry: registry,
	}, nil
}

func (m *LocalManager) RegisterModel(spec ModelSpec) error {
	return m.registry.Register(spec)
}

func (m *LocalManager) Devices(ctx context.Context) ([]Device, error) {
	return m.detector.Detect(ctx)
}

func (m *LocalManager) EnsureLoaded(ctx context.Context, modelID string, request SelectionRequest) (Backend, error) {
	devices, err := m.Devices(ctx)
	if err != nil {
		return "", err
	}

	backend, err := m.selector.Select(ctx, devices, request)
	if err != nil {
		return "", err
	}

	if err := m.registry.EnsureLoaded(ctx, modelID, backend); err != nil {
		return "", err
	}

	return backend, nil
}

func (m *LocalManager) Health(ctx context.Context) (Health, error) {
	devices, err := m.Devices(ctx)
	if err != nil {
		return Health{}, err
	}

	states := m.registry.States()
	notes := []string{"cpu fallback enabled"}
	ready := containsBackend(availableBackends(devices), BackendCPU)
	backend := BackendCPU

	for _, state := range states {
		if state.Loaded && state.Backend != "" && backend == BackendCPU {
			backend = state.Backend
		}
		if !state.Healthy {
			ready = false
		}
	}

	if len(states) == 0 {
		notes = append(notes, "no models registered")
	}

	return Health{
		Ready:   ready,
		Backend: backend,
		Notes:   notes,
		Devices: devices,
		Models:  states,
	}, nil
}

func (m *LocalManager) States() []ModelState {
	return m.registry.States()
}

type StaticManager struct {
	DetectedDevices []Device
	Status          Health
	Registered      []ModelState
}

func (m StaticManager) RegisterModel(ModelSpec) error {
	return nil
}

func (m StaticManager) EnsureLoaded(context.Context, string, SelectionRequest) (Backend, error) {
	if m.Status.Backend != "" {
		return m.Status.Backend, nil
	}
	return BackendCPU, nil
}

func (m StaticManager) Devices(context.Context) ([]Device, error) {
	if m.DetectedDevices == nil {
		return []Device{{Kind: DeviceCPU, Name: "cpu", Available: true}}, nil
	}
	return m.DetectedDevices, nil
}

func (m StaticManager) Health(context.Context) (Health, error) {
	if m.Status.Backend == "" {
		return Health{
			Ready:   true,
			Backend: BackendCPU,
			Notes:   []string{"cpu fallback enabled", "static runtime"},
			Devices: []Device{{Kind: DeviceCPU, Name: "cpu", Available: true}},
			Models:  append([]ModelState(nil), m.Registered...),
		}, nil
	}
	return m.Status, nil
}

func (m StaticManager) States() []ModelState {
	return append([]ModelState(nil), m.Registered...)
}

func availableBackends(devices []Device) []Backend {
	backends := make([]Backend, 0, len(devices))
	seen := map[Backend]struct{}{}
	for _, device := range devices {
		if !device.Available {
			continue
		}
		backend := device.Supports()
		if _, ok := seen[backend]; ok {
			continue
		}
		seen[backend] = struct{}{}
		backends = append(backends, backend)
	}
	return backends
}

func normalizeSupported(supported []Backend) []Backend {
	if len(supported) == 0 {
		return []Backend{BackendCPU, BackendCUDA, BackendDirectML, BackendCoreML, BackendOpenVINO}
	}
	return supported
}

func containsBackend(items []Backend, want Backend) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func isTruthy(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}
