package firequery

import (
	"context"
	"testing"

	"github.com/phmotad/firememory/internal/firequery/doctor"
	"github.com/phmotad/firememory/internal/firequery/mcp"
	fqruntime "github.com/phmotad/firememory/internal/firequery/runtime"
	"github.com/phmotad/firememory/internal/firequery/pipeline"
)

func TestNewRequiresDependencies(t *testing.T) {
	t.Parallel()

	server := mcp.NewServer()
	pipe := pipeline.NoopPipeline{}
	runtimeManager := fqruntime.StaticManager{}
	reporter := doctor.StaticReporter{}

	tests := []struct {
		name   string
		config Config
		want   error
	}{
		{
			name: "missing pipeline",
			config: Config{
				Runtime: runtimeManager,
				Doctor:  reporter,
				MCP:     server,
			},
			want: ErrPipelineRequired,
		},
		{
			name: "missing runtime",
			config: Config{
				Pipeline: pipe,
				Doctor:   reporter,
				MCP:      server,
			},
			want: ErrRuntimeRequired,
		},
		{
			name: "missing doctor",
			config: Config{
				Pipeline: pipe,
				Runtime:  runtimeManager,
				MCP:      server,
			},
			want: ErrDoctorRequired,
		},
		{
			name: "missing mcp",
			config: Config{
				Pipeline: pipe,
				Runtime:  runtimeManager,
				Doctor:   reporter,
			},
			want: ErrMCPRequired,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, err := New(tt.config)
			if err != tt.want {
				t.Fatalf("New() error = %v, want %v", err, tt.want)
			}
			if service != nil {
				t.Fatalf("expected nil service, got %#v", service)
			}
		})
	}
}

func TestNewBuildsService(t *testing.T) {
	t.Parallel()

	service, err := New(Config{
		Pipeline: pipeline.NoopPipeline{},
		Runtime:  fqruntime.StaticManager{},
		Doctor:   doctor.StaticReporter{},
		MCP:      mcp.NewServer(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if service.Pipeline() == nil || service.Runtime() == nil || service.Doctor() == nil || service.MCP() == nil {
		t.Fatal("service dependencies were not wired")
	}

	if _, err := service.Runtime().Devices(context.Background()); err != nil {
		t.Fatalf("Runtime().Devices() error = %v", err)
	}
}

func TestNewRejectsUnhealthyDoctorReport(t *testing.T) {
	t.Parallel()

	service, err := New(Config{
		Pipeline: pipeline.NoopPipeline{},
		Runtime:  fqruntime.StaticManager{},
		Doctor: doctor.StaticReporter{
			Report: doctor.Report{
				Ready: false,
				Checks: []doctor.Check{
					{Name: "intent-classifier", Status: "fail", Detail: "unhealthy"},
				},
			},
		},
		MCP: mcp.NewServer(),
	})
	if err != ErrNotReady {
		t.Fatalf("New() error = %v, want %v", err, ErrNotReady)
	}
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
}
