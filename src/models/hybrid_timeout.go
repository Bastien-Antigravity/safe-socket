package models

// =============================================================================
// ESSENTIAL PROCESS:
// Computes transport- and deployment-mode-aware hybrid idle timeouts and adaptive
// 2.5x heartbeat intervals for safe-socket connections:
// - SHM (Shared Memory): Infinite wait (Deadline=0, Heartbeat=0) with zero frame overhead.
// - UDP (Datagrams): Short datagram deadline (5s, Heartbeat=0).
// - TCP/TLS in Local Native Mode: Server=10m, Client=60s (auto 24s heartbeat).
// - TCP/TLS in Docker/Bridge Mode: Server=3m, Client=30s (auto 12s heartbeat to
//   keep NAT/conntrack state warm and reap crashed containers rapidly).
//
// DATA FLOW:
// 1. Input: Profile/transport name, socket role ("client"|"server"), and Docker mode flag.
// 2. Logic: Selects optimal bounded or infinite timeout profile for the transport & mode.
// 3. Output: Configured SocketConfig with Deadline and 2.5x HeartbeatInterval.
//
// KEY PARAMETERS:
// - profileName: Socket profile identifier ("tcp", "tcp-hello", "tls-hello", "shm", "udp").
// - socketRole: Connection role ("client" or "server").
// - isDocker: True when executing inside a containerized/bridge network (DOCKER_ENV=true).
// =============================================================================

import (
	"os"
	"strings"
	"time"
)

// -----------------------------------------------------------------------------

const (
	// LocalTCPServerIdleTimeout bounds zombie sockets on native loopback hosts.
	LocalTCPServerIdleTimeout = 10 * time.Minute
	// LocalTCPClientIdleTimeout drives a 24s (60s / 2.5) client heartbeat on native loopback hosts.
	LocalTCPClientIdleTimeout = 60 * time.Second
	// LocalTCPClientHeartbeat satisfies the 2.5x safety ratio for LocalTCPClientIdleTimeout.
	LocalTCPClientHeartbeat = 24 * time.Second

	// DockerTCPServerIdleTimeout reaps crashed container peers rapidly across Docker bridges/overlays.
	DockerTCPServerIdleTimeout = 3 * time.Minute
	// DockerTCPClientIdleTimeout drives a 12s (30s / 2.5) client heartbeat to prevent conntrack eviction.
	DockerTCPClientIdleTimeout = 30 * time.Second
	// DockerTCPClientHeartbeat satisfies the 2.5x safety ratio for DockerTCPClientIdleTimeout.
	DockerTCPClientHeartbeat = 12 * time.Second

	// DefaultUDPIdleTimeout bounds single-datagram UDP socket reads/writes.
	DefaultUDPIdleTimeout = 5 * time.Second
)

// -----------------------------------------------------------------------------

// IsDockerRuntime detects containerized execution from DOCKER_ENV or CONTAINER environment markers.
func IsDockerRuntime() bool {
	return os.Getenv("DOCKER_ENV") == "true" || os.Getenv("CONTAINER") == "true"
}

// -----------------------------------------------------------------------------

// ResolveHybridTimeouts returns the optimal (Deadline, HeartbeatInterval) pair for a given
// profile/transport name, socket role ("client" or "server"), and deployment mode (isDocker).
func ResolveHybridTimeouts(profileName string, socketRole string, isDocker bool) (time.Duration, time.Duration) {
	lowerProfile := strings.ToLower(strings.TrimSpace(profileName))
	isServer := strings.EqualFold(strings.TrimSpace(socketRole), "server")

	// 1. Shared Memory (SHM): zero kernel TCP half-open hazard -> pure infinite wait (0 overhead)
	if strings.HasPrefix(lowerProfile, "shm") {
		return 0, 0
	}

	// 2. Unreliable/Reliable Datagrams (UDP): bounded datagram timeout, zero background heartbeats
	if strings.HasPrefix(lowerProfile, "udp") {
		return DefaultUDPIdleTimeout, 0
	}

	// 3. Framed TCP / TLS persistent streams: mode-aware bounded timeout + 2.5x client heartbeat
	if isDocker {
		if isServer {
			return DockerTCPServerIdleTimeout, 0
		}
		return DockerTCPClientIdleTimeout, DockerTCPClientHeartbeat
	}

	if isServer {
		return LocalTCPServerIdleTimeout, 0
	}
	return LocalTCPClientIdleTimeout, LocalTCPClientHeartbeat
}

// -----------------------------------------------------------------------------

// NewHybridSocketConfig constructs a SocketConfig pre-populated with hybrid transport/mode timeouts
// and infinite reconnect resilience (MaxRetries: -1) for clients.
func NewHybridSocketConfig(profileName string, socketRole string, publicIP string, isDocker bool) SocketConfig {
	deadline, heartbeat := ResolveHybridTimeouts(profileName, socketRole, isDocker)
	maxRetries := 0
	if !strings.EqualFold(strings.TrimSpace(socketRole), "server") {
		maxRetries = -1
	}
	return SocketConfig{
		PublicIP:          publicIP,
		Deadline:          deadline,
		HeartbeatInterval: heartbeat,
		MaxRetries:        maxRetries,
	}
}
