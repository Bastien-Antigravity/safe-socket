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
// - SocketConfig: Struct holding PublicIP, ServiceAddress, Deadline, DisableTimeout, HeartbeatInterval, HandshakeTimeout.
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

	// Deadline is the default idle timeout for read/write operations.
	// If set to 0 and DisableTimeout is false, CreateWithConfig applies hybrid transport/mode defaults.
	// Set DisableTimeout: true (or Deadline < 0) to wait forever with zero idle deadlines.
	Deadline time.Duration

	// DisableTimeout explicitly disables idle read/write deadlines and automatic heartbeats
	// (equivalent to infinite wait / forever mode with Deadline = 0).
	DisableTimeout bool

	// HeartbeatInterval is the time between sending 0-length heartbeat frames.
	// For clients, 0 auto-calculates Deadline / 2.5; negative (< 0) disables heartbeats.
	// For servers, outbound heartbeats are disabled unless > 0 is explicitly provided.
	HeartbeatInterval time.Duration

	// HandshakeTimeout is the time allowed for the initial connection/protocol handshake.
	HandshakeTimeout time.Duration

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
