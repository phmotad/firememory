//go:build !onnx

package onnx

// New returns ErrNotAvailable when the binary is built without -tags onnx.
// Production binaries are always built with -tags onnx via `make build`.
func New(_ string) (Backend, error) {
	return nil, ErrNotAvailable
}
