package models

// SocketConfig holds runtime configuration for socket creation.
// This decouples static profile data (what we are) from runtime environment data (where we are).
type SocketConfig struct {
	// Application name
	Name string

	// Local host name, sockname
	Host string

	// Destination address
	Address string

	// PublicIP is the external IP address of this node, provided by the application.
	PublicIP string
}
