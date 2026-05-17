package daemon

import (
	"context"
	"fmt"
	"io"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

// startWriteWorker drains the write channel sequentially, running the full
// pipeline (ML inference + engine storage) for each job. Sequential execution
// ensures bbolt's single-writer constraint is never violated.
// Errors are written to errLog and do not stop the worker.
func startWriteWorker(
	ch <-chan writeJob,
	handleFn func(context.Context, contract.ExternalRequest) (contract.ExternalResponse, error),
	activity *activityMonitor,
	errLog io.Writer,
) {
	for job := range ch {
		_, err := handleFn(job.ctx, job.request)
		if err != nil {
			fmt.Fprintf(errLog, "daemon: async write error (op=%s req=%s): %v\n",
				job.request.Operation, job.request.RequestID, err)
		}
		activity.touch()
	}
}
