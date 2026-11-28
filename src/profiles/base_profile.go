package profiles

// =============================================================================
// ESSENTIAL PROCESS:
// Provides a shared BaseProfile implementation of interfaces.SocketProfile so all
// transport-specific profile structs inherit unified metadata and getters without duplication.
//
// DATA FLOW:
// 1. Input: Service identity name, endpoint address/path, transport type, protocol, and timeout.
// 2. Logic: Stores profile attributes and implements all five SocketProfile accessor methods.
// 3. Output: Reusable embedded struct for TCP, TLS, UDP, and SHM profiles.
//
// KEY PARAMETERS:
// - Name: Service identity string advertised during handshakes.
// - Address: Target host:port or SHM backing file path.
// - Transport: Underlying TransportType (FramedTCP, TLS, UDP, SharedMemory).
// - Protocol: Startup protocol (none or hello).
// - ConnectTimeout: Connection/handshake timeout in milliseconds.
// =============================================================================

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------

// BaseProfile implements interfaces.SocketProfile for all transport profiles.
type BaseProfile struct {
	Name           string
	Address        string
	Transport      interfaces.TransportType
	Protocol       interfaces.ProtocolType
	ConnectTimeout int
}

// GetName returns the unique identity name for this profile.
func (p *BaseProfile) GetName() string {
	return p.Name
}

// GetAddress returns the network address (IP:Port or SHM file path).
func (p *BaseProfile) GetAddress() string {
	return p.Address
}

// GetTransport returns the type of transport to use.
func (p *BaseProfile) GetTransport() interfaces.TransportType {
	return p.Transport
}

// GetProtocol returns the startup protocol to execute.
func (p *BaseProfile) GetProtocol() interfaces.ProtocolType {
	return p.Protocol
}

// GetConnectTimeout returns the timeout for establishing the connection in milliseconds.
func (p *BaseProfile) GetConnectTimeout() int {
	return p.ConnectTimeout
}
