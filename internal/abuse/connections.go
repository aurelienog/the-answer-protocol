package abuse

import (
	"net"
	"net/netip"
	"strings"
	"time"
)

type ipState struct {
	activeConnections int
	attempts          []time.Time
}

// ConnectionLease represents one admitted TCP connection.
type ConnectionLease struct {
	guard         *AbuseGuard
	ip            string
	authenticated bool
	released      bool
}

// AcquireConnection records an attempt even when one of the limits rejects it.
func (guard *AbuseGuard) AcquireConnection(remoteAddr net.Addr) (*ConnectionLease, ConnectionRejection) {
	if guard == nil {
		return nil, RejectGlobalConnections
	}

	ip := remoteIP(remoteAddr)
	now := guard.now()

	guard.mu.Lock()
	defer guard.mu.Unlock()

	state := guard.ipStates[ip]
	if state == nil {
		state = &ipState{}
		guard.ipStates[ip] = state
	}
	state.attempts = recent(state.attempts, now.Add(-guard.config.ConnectionAttemptWindow))

	if len(state.attempts) > guard.config.MaxConnectionAttemptsPerIP {
		return nil, RejectConnectionRate
	}
	state.attempts = append(state.attempts, now)
	if len(state.attempts) > guard.config.MaxConnectionAttemptsPerIP {
		return nil, RejectConnectionRate
	}
	if guard.activeConnections >= guard.config.MaxConnections {
		return nil, RejectGlobalConnections
	}
	if state.activeConnections >= guard.config.MaxConnectionsPerIP {
		return nil, RejectIPConnections
	}
	if guard.unauthenticatedConnections >= guard.config.MaxUnauthenticatedConnections {
		return nil, RejectUnauthenticated
	}

	state.activeConnections++
	guard.activeConnections++
	guard.unauthenticatedConnections++
	return &ConnectionLease{guard: guard, ip: ip}, ""
}

// Authenticate removes the connection from the unauthenticated count once.
func (lease *ConnectionLease) Authenticate() {
	if lease == nil || lease.guard == nil {
		return
	}
	guard := lease.guard
	guard.mu.Lock()
	defer guard.mu.Unlock()
	if lease.released || lease.authenticated {
		return
	}
	lease.authenticated = true
	guard.unauthenticatedConnections--
}

// Release frees all connection capacity held by this lease.
func (lease *ConnectionLease) Release() {
	if lease == nil || lease.guard == nil {
		return
	}
	guard := lease.guard
	guard.mu.Lock()
	defer guard.mu.Unlock()
	if lease.released {
		return
	}
	lease.released = true
	guard.activeConnections--
	if !lease.authenticated {
		guard.unauthenticatedConnections--
	}
	if state := guard.ipStates[lease.ip]; state != nil {
		state.activeConnections--
	}
}

func remoteIP(address net.Addr) string {
	if address == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(address.String())
	if err != nil {
		host = address.String()
	}
	host = strings.Trim(host, "[]")
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.Unmap().String()
	}
	return host
}
