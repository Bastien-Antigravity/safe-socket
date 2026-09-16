package models

// =============================================================================
// ESSENTIAL PROCESS:
// Encapsulates runtime environment configuration parameters for socket instances,
// decoupling profile templates from dynamic network attributes and connection timeouts.
//
// DATA FLOW:
// 1. Input: Service addresses, public IPs, deadline durations, and retry counts.
// 2. Logic: Injected into factory constructors and passed to transport decorators.
// 3. Output: Configured timeout deadlines, retry policies, and advertised identities.
//
// KEY PARAMETERS:
// - SocketConfig: Struct holding PublicIP, ServiceAddress, Deadline, HeartbeatInterval, HandshakeTimeout, MaxRetries.
// =============================================================================

import (
	"time"
)

// SocketConfig holds runtime configuration for socket creation.
// This decouples static profile data (what we are) from runtime environment data (where we are).
type SocketConfig struct {

	// PublicIP is the external IP address of this node, provided by the application.
	PublicIP string

	// ServiceAddress is the advertised inbound listening address of this service (e.g. "127.0.0.2:1026").
	// When provided, it is transmitted in the handshake FromAddress to enable dynamic service discovery.
	ServiceAddress string

	// Deadline is the default timeout for read/write operations on accepted connections.
	// If set to 0, no deadline is applied by default (server stays open/blocking).
	// This only applies to the *server* when accepting a new connection.
	Deadline time.Duration

	// HeartbeatInterval is the time between sending empty heartbeat frames (ping).
	// If 0, a default (e.g. 10s) may be used depending on the facade.
	HeartbeatInterval time.Duration

	// HandshakeTimeout is the time allowed for the initial protocol handshake.
	HandshakeTimeout time.Duration

	// MaxRetries is the number of times to attempt reconnection if Open() fails.
	// Set to -1 for infinite retries.
	MaxRetries int

	// RetryInterval is the time between reconnection attempts.
	RetryInterval time.Duration

	// Reliable enables the reliability layer for unreliable transports (UDP).
	// When enabled, packets will include sequence numbers and expect ACKs.
	Reliable bool

	// TLS Configuration
	CertFile           string
	KeyFile            string
	CAFile             string
	ServerName         string
	InsecureSkipVerify bool
}
