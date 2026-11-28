package profiles

// =============================================================================
// ESSENTIAL PROCESS:
// Defines UDP datagram socket profiles for connectionless or packet-sequenced
// low-latency network communication.
//
// DATA FLOW:
// 1. Input: Remote or local UDP address host:port and protocol type.
// 2. Logic: Embeds BaseProfile for UDP datagram transports.
// 3. Output: Configured UdpProfile instances.
//
// KEY PARAMETERS:
// - UdpProfile: Struct providing UDP transport metadata and protocol definitions.
// =============================================================================

import "github.com/Bastien-Antigravity/safe-socket/src/interfaces"

// -----------------------------------------------------------------------------
// UDP Profiles
// -----------------------------------------------------------------------------

type UdpProfile struct {
	BaseProfile
}

// -----------------------------------------------------------------------------

func NewUdpProfile(name, address string, timeout int) *UdpProfile {
	return &UdpProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportUDP,
			Protocol:       interfaces.ProtocolNone,
			ConnectTimeout: timeout,
		},
	}
}

func NewUdpHelloProfile(name, address string, timeout int) *UdpProfile {
	return &UdpProfile{
		BaseProfile: BaseProfile{
			Name:           name,
			Address:        address,
			Transport:      interfaces.TransportUDP,
			Protocol:       interfaces.ProtocolHello,
			ConnectTimeout: timeout,
		},
	}
}
