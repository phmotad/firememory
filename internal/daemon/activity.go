package daemon

import (
	"os"
	"sync/atomic"
	"time"
)

// activityMonitor tracks the last time the daemon received a request.
// If idle for longer than the configured timeout, it calls the shutdown
// function and exits the process so RAM is freed.
type activityMonitor struct {
	lastSeen atomic.Int64 // unix nanoseconds
}

func (m *activityMonitor) touch() {
	m.lastSeen.Store(time.Now().UnixNano())
}

func (m *activityMonitor) idleSince() time.Duration {
	last := m.lastSeen.Load()
	if last == 0 {
		return 0
	}
	return time.Since(time.Unix(0, last))
}

// watchAndExit runs in a goroutine. Every minute it checks whether the daemon
// has been idle for longer than idleTimeout. If so, shutdownFn is called
// (to flush pending writes and close the engine) before the process exits.
func (m *activityMonitor) watchAndExit(idleTimeout time.Duration, shutdownFn func()) {
	m.touch()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if m.idleSince() >= idleTimeout {
			shutdownFn()
			os.Exit(0)
		}
	}
}
