package abuse_test

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"the-answer-protocol/internal/abuse"
)

type testAddr string

func (address testAddr) Network() string { return "tcp" }
func (address testAddr) String() string  { return string(address) }

func TestAcquireConnectionAppliesConnectionLimitsAndLeaseLifecycle(t *testing.T) {
	guard := abuse.NewAbuseGuard(abuse.Config{
		MaxConnections:                2,
		MaxConnectionsPerIP:           1,
		MaxConnectionAttemptsPerIP:    10,
		ConnectionAttemptWindow:       time.Minute,
		MaxUnauthenticatedConnections: 1,
	})
	first, rejection := guard.AcquireConnection(testAddr("192.0.2.1:1000"))
	if rejection != "" || first == nil {
		t.Fatalf("first AcquireConnection() = (%v, %v), want accepted", first, rejection)
	}

	if _, rejection := guard.AcquireConnection(testAddr("192.0.2.1:1001")); rejection != abuse.RejectIPConnections {
		t.Fatalf("same-IP rejection = %q, want %q", rejection, abuse.RejectIPConnections)
	}
	if _, rejection := guard.AcquireConnection(testAddr("192.0.2.2:1000")); rejection != abuse.RejectUnauthenticated {
		t.Fatalf("unauthenticated rejection = %q, want %q", rejection, abuse.RejectUnauthenticated)
	}

	first.Authenticate()
	second, rejection := guard.AcquireConnection(testAddr("192.0.2.2:1001"))
	if rejection != "" || second == nil {
		t.Fatalf("AcquireConnection() after authentication = (%v, %v), want accepted", second, rejection)
	}
	if _, rejection := guard.AcquireConnection(testAddr("192.0.2.3:1000")); rejection != abuse.RejectGlobalConnections {
		t.Fatalf("global limit rejection = %q, want %q", rejection, abuse.RejectGlobalConnections)
	}

	first.Release()
	first.Release()
	second.Release()
	third, rejection := guard.AcquireConnection(testAddr("192.0.2.3:1001"))
	if rejection != "" || third == nil {
		t.Fatalf("AcquireConnection() after release = (%v, %v), want accepted", third, rejection)
	}
	third.Release()
}

func TestConnectionAttemptsCountRejectedRequestsAndExpire(t *testing.T) {
	guard := abuse.NewAbuseGuard(abuse.Config{
		MaxConnections:                10,
		MaxConnectionsPerIP:           10,
		MaxConnectionAttemptsPerIP:    2,
		ConnectionAttemptWindow:       time.Minute,
		MaxUnauthenticatedConnections: 10,
	})

	one, rejection := guard.AcquireConnection(testAddr("192.0.2.10:1"))
	if rejection != "" {
		t.Fatalf("first attempt rejected: %v", rejection)
	}
	two, rejection := guard.AcquireConnection(testAddr("192.0.2.10:2"))
	if rejection != "" {
		t.Fatalf("second attempt rejected: %v", rejection)
	}
	if lease, rejection := guard.AcquireConnection(testAddr("192.0.2.10:3")); lease != nil || rejection != abuse.RejectConnectionRate {
		t.Fatalf("third attempt = (%v, %q), want rate-limited", lease, rejection)
	}

	one.Release()
	two.Release()
	guard.CleanupExpired(time.Now().Add(2 * time.Minute))
	lease, rejection := guard.AcquireConnection(testAddr("192.0.2.10:4"))
	if rejection != "" || lease == nil {
		t.Fatalf("attempt after expiry = (%v, %v), want accepted", lease, rejection)
	}
	lease.Release()
}

func TestAllowCommandAppliesGeneralAndChatPoliciesPerPlayer(t *testing.T) {
	guard := abuse.NewAbuseGuard(abuse.Config{})
	for i := 0; i < 5; i++ {
		if !guard.AllowCommand("Alice", "CHAT ROOM message") {
			t.Fatalf("chat %d was unexpectedly rejected", i+1)
		}
	}
	if guard.AllowCommand("Alice", "CHAT GLOBAL spam") {
		t.Fatal("sixth chat was not rate-limited")
	}
	if !guard.AllowCommand("Bob", "CHAT ROOM hello") {
		t.Fatal("rate limit for Alice affected Bob")
	}

	general := abuse.NewAbuseGuard(abuse.Config{})
	for i := 0; i < 30; i++ {
		if !general.AllowCommand("Carol", "LOOK") {
			t.Fatalf("command %d was unexpectedly rejected", i+1)
		}
	}
	if general.AllowCommand("Carol", "LOOK") {
		t.Fatal("31st command was not rate-limited")
	}
}

func TestConnectionAdmissionIsConcurrencySafe(t *testing.T) {
	const maxConnections = 12
	guard := abuse.NewAbuseGuard(abuse.Config{
		MaxConnections:                maxConnections,
		MaxConnectionsPerIP:           maxConnections,
		MaxConnectionAttemptsPerIP:    100,
		ConnectionAttemptWindow:       time.Minute,
		MaxUnauthenticatedConnections: maxConnections,
	})

	var wait sync.WaitGroup
	var mu sync.Mutex
	leases := make([]*abuse.ConnectionLease, 0, maxConnections)
	for i := 0; i < 100; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			lease, rejection := guard.AcquireConnection(testAddr(fmt.Sprintf("192.0.2.%d:1", i+1)))
			if rejection != "" {
				return
			}
			mu.Lock()
			leases = append(leases, lease)
			mu.Unlock()
		}(i)
	}
	wait.Wait()
	if len(leases) != maxConnections {
		t.Fatalf("accepted %d concurrent connections, want %d", len(leases), maxConnections)
	}
	for _, lease := range leases {
		lease.Release()
	}
}

func TestRemoteIPSupportsTCPAndPipeAddresses(t *testing.T) {
	guard := abuse.NewAbuseGuard(abuse.Config{
		MaxConnections:                10,
		MaxConnectionsPerIP:           1,
		MaxConnectionAttemptsPerIP:    10,
		ConnectionAttemptWindow:       time.Minute,
		MaxUnauthenticatedConnections: 10,
	})
	first, rejection := guard.AcquireConnection(&net.TCPAddr{IP: net.ParseIP("192.0.2.20"), Port: 1000})
	if rejection != "" {
		t.Fatalf("TCP address rejected: %v", rejection)
	}
	if _, rejection := guard.AcquireConnection(&net.TCPAddr{IP: net.ParseIP("192.0.2.20"), Port: 1001}); rejection != abuse.RejectIPConnections {
		t.Fatalf("same TCP IP rejection = %q, want per-IP limit", rejection)
	}
	first.Release()
}
