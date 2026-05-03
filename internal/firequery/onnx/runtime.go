//go:build onnx

package onnx

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

const envOrtLibPath = "FIREMEMORY_ORT_LIB_PATH"

var (
	ortOnce sync.Once
	ortErr  error
)

// initORT initializes the ONNX Runtime environment exactly once per process.
// It looks for the shared library at FIREMEMORY_ORT_LIB_PATH, then next to the
// current executable, then relies on the OS dynamic linker (PATH / LD_LIBRARY_PATH).
func initORT() error {
	ortOnce.Do(func() {
		if path := os.Getenv(envOrtLibPath); path != "" {
			ort.SetSharedLibraryPath(path)
		} else if libPath := findOrtLibNextToExe(); libPath != "" {
			ort.SetSharedLibraryPath(libPath)
		}
		ortErr = ort.InitializeEnvironment()
	})
	return ortErr
}

func findOrtLibNextToExe() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(dir, "onnxruntime.dll"),
		filepath.Join(dir, "libonnxruntime.so"),
		filepath.Join(dir, "libonnxruntime.dylib"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// newDynamicSession creates a dynamic-shape ONNX session for the given model file.
func newDynamicSession(modelPath string, inputNames, outputNames []string) (*ort.DynamicAdvancedSession, error) {
	if err := initORT(); err != nil {
		return nil, fmt.Errorf("onnx: ORT init: %w", err)
	}
	opts, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("onnx: session options: %w", err)
	}
	defer opts.Destroy()
	return ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, opts)
}
