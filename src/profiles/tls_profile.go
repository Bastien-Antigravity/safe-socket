package profiles

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------
// TLS Client Profile
// -----------------------------------------------------------------------------

type TlsClientProfile struct {
	Name    string
	Address string
	Timeout int
}

func NewTlsClientProfile(name, address string, timeout int) *TlsClientProfile {
	return &TlsClientProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

func (p *TlsClientProfile) GetName() string                     { return p.Name }
func (p *TlsClientProfile) GetAddress() string                  { return p.Address }
func (p *TlsClientProfile) GetTransport() interfaces.TransportType { return interfaces.TransportTLS }
func (p *TlsClientProfile) GetProtocol() interfaces.ProtocolType  { return interfaces.ProtocolNone }
func (p *TlsClientProfile) GetConnectTimeout() int              { return p.Timeout }

// -----------------------------------------------------------------------------
// TLS Hello Client Profile
// -----------------------------------------------------------------------------

type TlsHelloClientProfile struct {
	Name    string
	Address string
	Timeout int
}

func NewTlsHelloClientProfile(name, address string, timeout int) *TlsHelloClientProfile {
	return &TlsHelloClientProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

func (p *TlsHelloClientProfile) GetName() string                     { return p.Name }
func (p *TlsHelloClientProfile) GetAddress() string                  { return p.Address }
func (p *TlsHelloClientProfile) GetTransport() interfaces.TransportType { return interfaces.TransportTLS }
func (p *TlsHelloClientProfile) GetProtocol() interfaces.ProtocolType  { return interfaces.ProtocolHello }
func (p *TlsHelloClientProfile) GetConnectTimeout() int              { return p.Timeout }

// -----------------------------------------------------------------------------
// TLS Server Profile
// -----------------------------------------------------------------------------

type TlsServerProfile struct {
	Name    string
	Address string
	Timeout int
}

func NewTlsServerProfile(name, address string, timeout int) *TlsServerProfile {
	return &TlsServerProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

func (p *TlsServerProfile) GetName() string                     { return p.Name }
func (p *TlsServerProfile) GetAddress() string                  { return p.Address }
func (p *TlsServerProfile) GetTransport() interfaces.TransportType { return interfaces.TransportTLS }
func (p *TlsServerProfile) GetProtocol() interfaces.ProtocolType  { return interfaces.ProtocolNone }
func (p *TlsServerProfile) GetConnectTimeout() int              { return p.Timeout }

// -----------------------------------------------------------------------------
// TLS Hello Server Profile
// -----------------------------------------------------------------------------

type TlsHelloServerProfile struct {
	Name    string
	Address string
	Timeout int
}

func NewTlsHelloServerProfile(name, address string, timeout int) *TlsHelloServerProfile {
	return &TlsHelloServerProfile{
		Name:    name,
		Address: address,
		Timeout: timeout,
	}
}

func (p *TlsHelloServerProfile) GetName() string                     { return p.Name }
func (p *TlsHelloServerProfile) GetAddress() string                  { return p.Address }
func (p *TlsHelloServerProfile) GetTransport() interfaces.TransportType { return interfaces.TransportTLS }
func (p *TlsHelloServerProfile) GetProtocol() interfaces.ProtocolType  { return interfaces.ProtocolHello }
func (p *TlsHelloServerProfile) GetConnectTimeout() int              { return p.Timeout }
