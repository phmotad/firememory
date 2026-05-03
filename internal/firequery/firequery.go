package firequery

import (
	"context"
	"errors"
	"fmt"

	"github.com/phmotad/firememory/internal/firequery/doctor"
	fqmcp "github.com/phmotad/firememory/internal/firequery/mcp"
	fqruntime "github.com/phmotad/firememory/internal/firequery/runtime"
	"github.com/phmotad/firememory/internal/firequery/pipeline"
)

var (
	ErrPipelineRequired = errors.New("firequery: pipeline is required")
	ErrRuntimeRequired  = errors.New("firequery: runtime is required")
	ErrDoctorRequired   = errors.New("firequery: doctor is required")
	ErrMCPRequired      = errors.New("firequery: mcp server is required")
	ErrNotReady         = errors.New("firequery: mandatory specialists are not healthy")
)

type Config struct {
	Pipeline pipeline.Pipeline
	Runtime  fqruntime.Manager
	Doctor   doctor.Reporter
	MCP      *fqmcp.Server
}

type Service struct {
	pipeline pipeline.Pipeline
	runtime  fqruntime.Manager
	doctor   doctor.Reporter
	mcp      *fqmcp.Server
}

func New(config Config) (*Service, error) {
	if config.Pipeline == nil {
		return nil, ErrPipelineRequired
	}
	if config.Runtime == nil {
		return nil, ErrRuntimeRequired
	}
	if config.Doctor == nil {
		return nil, ErrDoctorRequired
	}
	if config.MCP == nil {
		return nil, ErrMCPRequired
	}
	report, err := config.Doctor.Run(context.Background())
	if err != nil {
		return nil, fmt.Errorf("firequery: doctor check failed: %w", err)
	}
	if !report.Ready {
		return nil, ErrNotReady
	}

	return &Service{
		pipeline: config.Pipeline,
		runtime:  config.Runtime,
		doctor:   config.Doctor,
		mcp:      config.MCP,
	}, nil
}

func (s *Service) Pipeline() pipeline.Pipeline {
	return s.pipeline
}

func (s *Service) Runtime() fqruntime.Manager {
	return s.runtime
}

func (s *Service) Doctor() doctor.Reporter {
	return s.doctor
}

func (s *Service) MCP() *fqmcp.Server {
	return s.mcp
}
