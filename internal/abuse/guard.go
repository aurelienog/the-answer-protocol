package abuse

import (
	"sync"
	"time"
)

// AbuseGuard coordinates connection admission and per-player rate limits.
type AbuseGuard struct {
	mu                         sync.Mutex
	config                     Config
	ipStates                   map[string]*ipState
	rateEvents                 map[string][]time.Time
	activeConnections          int
	unauthenticatedConnections int
	now                        func() time.Time
}

// NewAbuseGuard creates a guard using the supplied limits.
func NewAbuseGuard(config Config) *AbuseGuard {
	defaults := DefaultConfig()
	if config.MaxConnections <= 0 {
		config.MaxConnections = defaults.MaxConnections
	}
	if config.MaxConnectionsPerIP <= 0 {
		config.MaxConnectionsPerIP = defaults.MaxConnectionsPerIP
	}
	if config.MaxConnectionAttemptsPerIP <= 0 {
		config.MaxConnectionAttemptsPerIP = defaults.MaxConnectionAttemptsPerIP
	}
	if config.ConnectionAttemptWindow <= 0 {
		config.ConnectionAttemptWindow = defaults.ConnectionAttemptWindow
	}
	if config.MaxUnauthenticatedConnections <= 0 {
		config.MaxUnauthenticatedConnections = defaults.MaxUnauthenticatedConnections
	}
	return &AbuseGuard{
		config:     config,
		ipStates:   make(map[string]*ipState),
		rateEvents: make(map[string][]time.Time),
		now:        time.Now,
	}
}
