package devices

import (
	"context"

	fqruntime "github.com/phmotad/firememory/internal/firequery/runtime"
)

type Manager interface {
	Devices(ctx context.Context) ([]fqruntime.Device, error)
	Health(ctx context.Context) (fqruntime.Health, error)
	RegisterModel(spec fqruntime.ModelSpec) error
	EnsureLoaded(ctx context.Context, modelID string, request fqruntime.SelectionRequest) (fqruntime.Backend, error)
	States() []fqruntime.ModelState
}

type GoDeviceManager struct {
	runtime fqruntime.Manager
}

func NewGoDeviceManager(runtime fqruntime.Manager) GoDeviceManager {
	return GoDeviceManager{runtime: runtime}
}

func (m GoDeviceManager) Devices(ctx context.Context) ([]fqruntime.Device, error) {
	return m.runtime.Devices(ctx)
}

func (m GoDeviceManager) Health(ctx context.Context) (fqruntime.Health, error) {
	return m.runtime.Health(ctx)
}

func (m GoDeviceManager) RegisterModel(spec fqruntime.ModelSpec) error {
	return m.runtime.RegisterModel(spec)
}

func (m GoDeviceManager) EnsureLoaded(ctx context.Context, modelID string, request fqruntime.SelectionRequest) (fqruntime.Backend, error) {
	return m.runtime.EnsureLoaded(ctx, modelID, request)
}

func (m GoDeviceManager) States() []fqruntime.ModelState {
	return m.runtime.States()
}
