package abuse

import "time"

// Config contains the resource and request limits enforced by AbuseGuard.
type Config struct {
	MaxConnections                int
	MaxConnectionsPerIP           int
	MaxConnectionAttemptsPerIP    int
	ConnectionAttemptWindow       time.Duration
	MaxUnauthenticatedConnections int
}

// DefaultConfig returns conservative defaults for a small shared game server.
func DefaultConfig() Config {
	return Config{
		MaxConnections:                100,
		MaxConnectionsPerIP:           5,
		MaxConnectionAttemptsPerIP:    10,
		ConnectionAttemptWindow:       time.Minute,
		MaxUnauthenticatedConnections: 20,
	}
}

const (
	maxCommandsPerPlayer = 30
	commandWindow        = 10 * time.Second
	maxChatsPerPlayer    = 5
	chatWindow           = 10 * time.Second
)

// ConnectionRejection describes why a new connection was rejected.
type ConnectionRejection string

func (reason ConnectionRejection) Error() string {
	return string(reason)
}

const (
	RejectGlobalConnections ConnectionRejection = "global connection limit reached"
	RejectIPConnections     ConnectionRejection = "per-IP connection limit reached"
	RejectConnectionRate    ConnectionRejection = "connection attempt rate limit reached"
	RejectUnauthenticated   ConnectionRejection = "unauthenticated connection limit reached"
)
