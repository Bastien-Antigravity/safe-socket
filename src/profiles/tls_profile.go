package profiles

// =============================================================================
// ESSENTIAL PROCESS:
// Defines TLS socket profiles for encrypted client and server stream transports,
// securing network communication across untrusted network boundaries.
//
// DATA FLOW:
// 1. Input: Target endpoint address, timeouts, and cryptographic options.
// 2. Logic: Embeds BaseProfile specifying TLS transport properties.
// 3. Output: TlsClientProfile, TlsHelloClientProfile, TlsServerProfile, and TlsHelloServerProfile instances.
//
// KEY PARAMETERS:
// - TlsClientProfile: Outbound TLS client profile.
// - TlsServerProfile: Inbound TLS listener profile.
// =============================================================================

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------
// TLS Client Profiles
// -----------------------------------------------------------------------------

type TlsClientProfile struct {
	BaseProfile
}

func NewTlsClientProfile(name, address string, timeout int) *TlsClientProfile {
	return &TlsClientProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportTLS,
			Protocol:       interfaces.ProtocolNone,
			ConnectTimeout: timeout,
		},
	}
}

// -----------------------------------------------------------------------------

type TlsHelloClientProfile struct {
	BaseProfile
}

func NewTlsHelloClientProfile(name, address string, timeout int) *TlsHelloClientProfile {
	return &TlsHelloClientProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportTLS,
			Protocol:       interfaces.ProtocolHello,
			ConnectTimeout: timeout,
		},
	}
}

// -----------------------------------------------------------------------------
// TLS Server Profiles
// -----------------------------------------------------------------------------

type TlsServerProfile struct {
	BaseProfile
}

func NewTlsServerProfile(name, address string, timeout int) *TlsServerProfile {
	return &TlsServerProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportTLS,
			Protocol:       interfaces.ProtocolNone,
			ConnectTimeout: timeout,
		},
	}
}

// -----------------------------------------------------------------------------

type TlsHelloServerProfile struct {
	BaseProfile
}

func NewTlsHelloServerProfile(name, address string, timeout int) *TlsHelloServerProfile {
	return &TlsHelloServerProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportTLS,
			Protocol:       interfaces.ProtocolHello,
			ConnectTimeout: timeout,
		},
	}
}
