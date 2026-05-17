package daemon

import (
	"context"

	"github.com/phmotad/firememory/internal/firequery/contract"
)

// writeJob carries an async write request through the daemon's write channel.
type writeJob struct {
	ctx     context.Context
	request contract.ExternalRequest
}
