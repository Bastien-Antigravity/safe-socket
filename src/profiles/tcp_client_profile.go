package profiles

// =============================================================================
// ESSENTIAL PROCESS:
// Defines TCP client socket profiles for standard stream-oriented connections
// with or without application-level handshake protocols.
//
// DATA FLOW:
// 1. Input: Endpoint host:port strings, timeouts, and protocol specifications.
// 2. Logic: Embeds BaseProfile to declare TCP transport characteristics for clients.
// 3. Output: TcpClientProfile and TcpHelloClientProfile configurations.
//
// KEY PARAMETERS:
// - TcpClientProfile: Profile for raw framed TCP communication.
// - TcpHelloClientProfile: Profile enforcing Cap'n Proto HelloMsg handshake.
// =============================================================================

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------
// TCP Client Profile (Raw)
// -----------------------------------------------------------------------------

// TcpClientProfile implements the SocketProfile interface for a raw TCP connection
// without any handshake protocol.
type TcpClientProfile struct {
	BaseProfile
}

// NewTcpClientProfile creates a new instance of a TCP profile without a protocol.
func NewTcpClientProfile(name, address string, timeout int) *TcpClientProfile {
	return &TcpClientProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportFramedTCP,
			Protocol:       interfaces.ProtocolNone,
			ConnectTimeout: timeout,
		},
	}
}

// -----------------------------------------------------------------------------
// TCP Hello Client Profile
// -----------------------------------------------------------------------------

// TcpHelloClientProfile implements the SocketProfile interface for a TCP connection
// that performs a handshake protocol upon connection.
type TcpHelloClientProfile struct {
	BaseProfile
}

// NewTcpHelloClientProfile creates a new instance of a TCP profile with the Hello protocol.
func NewTcpHelloClientProfile(name, address string, timeout int) *TcpHelloClientProfile {
	return &TcpHelloClientProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportFramedTCP,
			Protocol:       interfaces.ProtocolHello,
			ConnectTimeout: timeout,
		},
	}
}
