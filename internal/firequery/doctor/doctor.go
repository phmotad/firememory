package doctor

import (
	"context"
	"fmt"

	fqruntime "github.com/phmotad/firememory/internal/firequery/runtime"
)

type Check struct {
	Name   string
	Status string
	Detail string
}

type Report struct {
	Ready  bool
	Checks []Check
}

type Reporter interface {
	Run(ctx context.Context) (Report, error)
}

type StaticReporter struct {
	Report Report
}

func (r StaticReporter) Run(context.Context) (Report, error) {
	if !r.Report.Ready && len(r.Report.Checks) == 0 {
		return Report{
			Ready: true,
			Checks: []Check{
				{Name: "runtime", Status: "ok", Detail: "static reporter"},
			},
		}, nil
	}
	return r.Report, nil
}

type RuntimeReporter struct {
	Runtime fqruntime.Manager
}

func (r RuntimeReporter) Run(ctx context.Context) (Report, error) {
	health, err := r.Runtime.Health(ctx)
	if err != nil {
		return Report{}, err
	}

	checks := make([]Check, 0, 2+len(health.Models))
	checks = append(checks, Check{
		Name:   "runtime",
		Status: statusOf(health.Ready),
		Detail: fmt.Sprintf("backend=%s", health.Backend),
	})
	checks = append(checks, Check{
		Name:   "devices",
		Status: statusOf(len(health.Devices) > 0),
		Detail: fmt.Sprintf("detected=%d", len(health.Devices)),
	})

	for _, model := range health.Models {
		detail := fmt.Sprintf("backend=%s loaded=%t", model.Backend, model.Loaded)
		if len(model.Notes) > 0 {
			detail += " notes=" + model.Notes[0]
		}
		checks = append(checks, Check{
			Name:   "model:" + model.ID,
			Status: statusOf(model.Healthy),
			Detail: detail,
		})
	}

	return Report{
		Ready:  health.Ready,
		Checks: checks,
	}, nil
}

func statusOf(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}
