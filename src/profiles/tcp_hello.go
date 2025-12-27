package profiles

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// TcpHelloProfile implements the SocketProfile interface for a TCP connection
// that performs a handshake protocol upon connection.
type TcpHelloProfile struct {
	Name    string
	Address string
	Timeout int
}

// -----------------------------------------------------------------------------

// NewTcpHelloProfile creates a new instance of a TCP profile with the Hello protocol.
func NewTcpHelloProfile(name, address string, timeout int) *TcpHelloProfile {
	return &TcpHelloProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

// -----------------------------------------------------------------------------

// GetName returns the name assigned to this profile.
func (p *TcpHelloProfile) GetName() string {
	return p.Name
}

// -----------------------------------------------------------------------------

// GetAddress returns the target network address.
func (p *TcpHelloProfile) GetAddress() string {
	return p.Address
}

// -----------------------------------------------------------------------------

// GetTransport specifies the use of Framed TCP for this profile.
func (p *TcpHelloProfile) GetTransport() interfaces.TransportType {
	return interfaces.TransportFramedTCP
}

// -----------------------------------------------------------------------------

// GetProtocol specifies the use of the SayHello handshake protocol.
func (p *TcpHelloProfile) GetProtocol() interfaces.ProtocolType {
	return interfaces.ProtocolSayHello
}

// -----------------------------------------------------------------------------

// GetConnectTimeout returns the configured connection timeout.
func (p *TcpHelloProfile) GetConnectTimeout() int {
	return p.Timeout
}
