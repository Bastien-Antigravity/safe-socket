package transports

// =============================================================================
// ESSENTIAL PROCESS:
// Implements client connection establishment for UDP datagram transports.
//
// DATA FLOW:
// 1. Input: Remote UDP endpoint address and idle timeout.
// 2. Logic: Resolves UDP address and executes net.DialUDP to set default peer destination.
// 3. Output: Initialized UdpSocket as interfaces.TransportConnection.
//
// KEY PARAMETERS:
// - ConnectUDP: Dialer function for client UDP sockets.
// =============================================================================

import (
	"net"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// -----------------------------------------------------------------------------

// ConnectUDP creates a UDP connection.
// Note: UDP is connectionless. "Dial" just sets the default destination address.
func ConnectUDP(address string, timeout time.Duration) (interfaces.TransportConnection, error) {
	// Resolve address
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	// Dial (connect) to setting the default remote address
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	// Reliability: Increase OS buffers to reduce packet drops
	// 4MB buffers (adjust based on needs/OS limits)
	_ = conn.SetReadBuffer(4 * 1024 * 1024)
	_ = conn.SetWriteBuffer(4 * 1024 * 1024)

	return NewUdpSocket(conn, timeout), nil
}
