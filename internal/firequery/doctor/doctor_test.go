package doctor

import (
	"context"
	"testing"

	fqruntime "github.com/phmotad/firememory/internal/firequery/runtime"
)

func TestRuntimeReporter(t *testing.T) {
	t.Parallel()

	reporter := RuntimeReporter{
		Runtime: fqruntime.StaticManager{
			Status: fqruntime.Health{
				Ready:   true,
				Backend: fqruntime.BackendCPU,
				Devices: []fqruntime.Device{{Kind: fqruntime.DeviceCPU, Name: "cpu", Available: true}},
				Models: []fqruntime.ModelState{{
					ID:      "intent",
					Backend: fqruntime.BackendCPU,
					Loaded:  true,
					Healthy: true,
				}},
			},
		},
	}

	report, err := reporter.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.Ready {
		t.Fatal("report should be ready")
	}
	if len(report.Checks) < 3 {
		t.Fatalf("checks = %d, want >= 3", len(report.Checks))
	}
}
