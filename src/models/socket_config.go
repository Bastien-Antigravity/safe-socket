package models

import (
	"time"
)

// SocketConfig holds runtime configuration for socket creation.
// This decouples static profile data (what we are) from runtime environment data (where we are).
type SocketConfig struct {

	// PublicIP is the external IP address of this node, provided by the application.
	PublicIP string

	// Deadline is the default timeout for read/write operations on accepted connections.
	// If set to 0, no deadline is applied by default (server stays open/blocking).
	// This only applies to the *server* when accepting a new connection.
	Deadline time.Duration
}
