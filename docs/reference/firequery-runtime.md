# FireQuery Runtime

## Goal

The runtime selects and operates specialist components on the local machine.

It must work offline for tests and support CPU fallback in production.

## Components

### DeviceDetector

Detects available execution targets such as:

- CPU
- CUDA
- DirectML
- CoreML
- OpenVINO

### BackendSelector

Chooses the backend for each specialist according to:

- availability
- model support
- memory budget
- latency target
- explicit user or config preference

### ModelRegistry

Tracks:

- model id
- version
- backend
- load state
- health state

### HealthChecker

Runs lightweight checks to ensure a specialist is callable.

### DeviceManager

The device manager is implemented in Go and owns:

- device detection
- backend selection
- model state tracking
- health visibility
- CPU fallback

## Runtime rules

- CPU fallback is mandatory.
- No specialist should block the whole system if an optional accelerator is unavailable.
- Lazy loading is preferred over eager startup.
- Health status must be visible to diagnostics.
- Runtime failures should degrade gracefully when possible.

## Diagnostics

Planned commands:

- `fquery devices`
- `fquery doctor`

`fquery devices` should report detected hardware and available backends.

`fquery doctor` should report specialist readiness and runtime health.

## Testability

- Device detection must be mockable.
- Backend selection must be deterministic under test.
- Health checks must support fake implementations.
- FireQuery tests must pass on CPU-only machines.
