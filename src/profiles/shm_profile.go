package profiles

// =============================================================================
// ESSENTIAL PROCESS:
// Defines shared-memory (SHM) socket profiles for high-throughput, low-latency
// intra-node IPC communications, decoupling service identity from backing file paths.
//
// DATA FLOW:
// 1. Input: Service identity name, memory-mapped file paths, timeouts, and protocol types.
// 2. Logic: Fulfills the SocketProfile interface for shared-memory transport resolution.
// 3. Output: Configured SHM profile objects for socket factory instantiation.
//
// KEY PARAMETERS:
// - ShmProfile: Struct specifying SHM service identity, file path, and handshake protocol.
// =============================================================================

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------
// SHM Profiles
// -----------------------------------------------------------------------------

// ShmProfile implements interfaces.SocketProfile for Shared Memory (SHM) IPC.
type ShmProfile struct {
	BaseProfile
}

// GetAddress returns the backing SHM file path, falling back to Name if constructed directly without Address.
func (p *ShmProfile) GetAddress() string {
	if p.Address != "" {
		return p.Address
	}
	return p.Name
}

// -----------------------------------------------------------------------------

// NewShmProfile creates a raw SHM profile using path as both identity and file path.
func NewShmProfile(path string, timeout int) *ShmProfile {
	return NewShmProfileWithIdentity(path, path, timeout)
}

// NewShmHelloProfile creates a Hello-enabled SHM profile using path as both identity and file path.
func NewShmHelloProfile(path string, timeout int) *ShmProfile {
	return NewShmHelloProfileWithIdentity(path, path, timeout)
}

// NewShmProfileWithIdentity creates a raw SHM profile with distinct identity name and file path.
func NewShmProfileWithIdentity(name, path string, timeout int) *ShmProfile {
	return &ShmProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        path,
			Transport:      interfaces.TransportShm,
			Protocol:       interfaces.ProtocolNone,
			ConnectTimeout: timeout,
		},
	}
}

// NewShmHelloProfileWithIdentity creates a Hello-enabled SHM profile with distinct identity name and file path.
func NewShmHelloProfileWithIdentity(name, path string, timeout int) *ShmProfile {
	return &ShmProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        path,
			Transport:      interfaces.TransportShm,
			Protocol:       interfaces.ProtocolHello,
			ConnectTimeout: timeout,
		},
	}
}
