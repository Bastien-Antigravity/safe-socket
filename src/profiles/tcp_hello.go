package profiles

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// TcpHelloProfile is a pre-configured profile for TCP transport with Hello protocol.
type TcpHelloProfile struct {
	Name    string
	Address string
	Timeout int
}

func NewTcpHelloProfile(name, address string, timeout int) *TcpHelloProfile {
	return &TcpHelloProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

func (p *TcpHelloProfile) GetName() string {
	return p.Name
}

func (p *TcpHelloProfile) GetAddress() string {
	return p.Address
}

func (p *TcpHelloProfile) GetTransport() interfaces.TransportType {
	return interfaces.TransportFramedTCP
}

func (p *TcpHelloProfile) GetProtocol() interfaces.ProtocolType {
	return interfaces.ProtocolSayHello
}

func (p *TcpHelloProfile) GetConnectTimeout() int {
	return p.Timeout
}
