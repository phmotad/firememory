package runtime

import (
	"context"
	"errors"
	"testing"
)

func TestEnvironmentDetectorDetect(t *testing.T) {
	t.Parallel()

	detector := EnvironmentDetector{
		LookupEnv: func(key string) string {
			switch key {
			case "FIREQUERY_ENABLE_CUDA", "FIREQUERY_ENABLE_OPENVINO":
				return "1"
			default:
				return ""
			}
		},
	}

	devices, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(devices) != 3 {
		t.Fatalf("len(devices) = %d, want 3", len(devices))
	}
}

func TestLocalBackendSelector(t *testing.T) {
	t.Parallel()

	selector := LocalBackendSelector{}
	devices := []Device{
		{Kind: DeviceCPU, Name: "cpu", Available: true},
		{Kind: DeviceCUDA, Name: "cuda", Available: true},
	}

	backend, err := selector.Select(context.Background(), devices, SelectionRequest{
		ModelID:    "intent",
		Preferred:  BackendCUDA,
		RequireGPU: false,
		Supported:  []Backend{BackendCPU, BackendCUDA},
	})
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if backend != BackendCUDA {
		t.Fatalf("backend = %q, want cuda", backend)
	}

	backend, err = selector.Select(context.Background(), []Device{{Kind: DeviceCPU, Name: "cpu", Available: true}}, SelectionRequest{
		ModelID:   "intent",
		Supported: []Backend{BackendCPU},
	})
	if err != nil {
		t.Fatalf("Select() fallback error = %v", err)
	}
	if backend != BackendCPU {
		t.Fatalf("backend = %q, want cpu", backend)
	}
}

func TestRegistryLazyLoadingAndBudget(t *testing.T) {
	t.Parallel()

	loads := 0
	registry := NewRegistry(256)
	err := registry.Register(ModelSpec{
		ID:          "intent",
		Version:     "0.1",
		Backends:    []Backend{BackendCPU},
		MemoryBytes: 64,
		Loader: func(context.Context, Backend) error {
			loads++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := registry.EnsureLoaded(context.Background(), "intent", BackendCPU); err != nil {
		t.Fatalf("EnsureLoaded() error = %v", err)
	}
	if err := registry.EnsureLoaded(context.Background(), "intent", BackendCPU); err != nil {
		t.Fatalf("EnsureLoaded() second error = %v", err)
	}
	if loads != 1 {
		t.Fatalf("loads = %d, want 1", loads)
	}

	err = registry.Register(ModelSpec{
		ID:          "reranker",
		Version:     "0.1",
		Backends:    []Backend{BackendCPU},
		MemoryBytes: 500,
		Loader: func(context.Context, Backend) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Register() second error = %v", err)
	}
	if err := registry.EnsureLoaded(context.Background(), "reranker", BackendCPU); !errors.Is(err, ErrMemoryBudgetExceeded) {
		t.Fatalf("EnsureLoaded() budget error = %v, want %v", err, ErrMemoryBudgetExceeded)
	}
}

func TestManagerHealth(t *testing.T) {
	t.Parallel()

	registry := NewRegistry(128)
	if err := registry.Register(ModelSpec{
		ID:          "intent",
		Version:     "0.1",
		Backends:    []Backend{BackendCPU},
		MemoryBytes: 32,
		Loader: func(context.Context, Backend) error {
			return nil
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	manager, err := NewManager(
		EnvironmentDetector{LookupEnv: func(string) string { return "" }},
		LocalBackendSelector{},
		registry,
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if _, err := manager.EnsureLoaded(context.Background(), "intent", SelectionRequest{
		ModelID:   "intent",
		Supported: []Backend{BackendCPU},
	}); err != nil {
		t.Fatalf("EnsureLoaded() error = %v", err)
	}

	health, err := manager.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if !health.Ready {
		t.Fatalf("Health().Ready = false, want true")
	}
	if len(health.Models) != 1 || !health.Models[0].Loaded {
		t.Fatalf("Health().Models = %#v", health.Models)
	}
}
