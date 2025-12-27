package profiles

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// TcpProfile implements the SocketProfile interface for a raw TCP connection
// without any handshake protocol.
type TcpProfile struct {
	Name    string
	Address string
	Timeout int
}

// -----------------------------------------------------------------------------

// NewTcpProfile creates a new instance of a TCP profile without a protocol.
func NewTcpProfile(name, address string, timeout int) *TcpProfile {
	return &TcpProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

// -----------------------------------------------------------------------------

// GetName returns the name assigned to this profile.
func (p *TcpProfile) GetName() string {
	return p.Name
}

// -----------------------------------------------------------------------------

// GetAddress returns the target network address.
func (p *TcpProfile) GetAddress() string {
	return p.Address
}

// -----------------------------------------------------------------------------

// GetTransport specifies the use of Framed TCP for this profile.
func (p *TcpProfile) GetTransport() interfaces.TransportType {
	return interfaces.TransportFramedTCP
}

// -----------------------------------------------------------------------------

// GetProtocol specifies that no handshake protocol is used.
func (p *TcpProfile) GetProtocol() interfaces.ProtocolType {
	return interfaces.ProtocolNone
}

// -----------------------------------------------------------------------------

// GetConnectTimeout returns the configured connection timeout.
func (p *TcpProfile) GetConnectTimeout() int {
	return p.Timeout
}
