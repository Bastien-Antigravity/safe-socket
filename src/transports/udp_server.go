package transports

import (
	"net"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// UdpListener implements interfaces.TransportListener for Connectionless UDP.
type UdpListener struct {
	Conn    *net.UDPConn
	Timeout time.Duration
}

// -----------------------------------------------------------------------------

func ListenUDP(address string, timeout time.Duration) (interfaces.TransportListener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	// Optimizations (match client)
	_ = conn.SetReadBuffer(4 * 1024 * 1024)
	_ = conn.SetWriteBuffer(4 * 1024 * 1024)

	return &UdpListener{
		Conn:    conn,
		Timeout: timeout,
	}, nil
}

// -----------------------------------------------------------------------------

// Accept "accepts" the next packet as if it were a new connection.
// It blocks until a packet arrives, reads it, and returns a TransientUdpSocket
// bound to that sender.
func (l *UdpListener) Accept() (interfaces.TransportConnection, error) {
	// Buffer for the initial packet (Accept reads it to identify sender)
	// 4KB should be enough for control frames / hello messages.
	buf := make([]byte, 4096)

	if l.Timeout > 0 {
		_ = l.Conn.SetReadDeadline(time.Now().Add(l.Timeout))
	}

	// ReadFromUDP to get data AND sender address
	n, remoteAddr, err := l.Conn.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}

	// Create a Transient Socket wrapping this specific packet interaction
	// The next Read() on this socket will return 'buf[:n]'.
	// The next Write() on this socket will send to 'remoteAddr'.
	return NewTransientUdpSocket(l.Conn, remoteAddr, buf[:n], l.Timeout), nil
}

// -----------------------------------------------------------------------------

func (l *UdpListener) Close() error {
	return l.Conn.Close()
}

// -----------------------------------------------------------------------------

func (l *UdpListener) Addr() net.Addr {
	return l.Conn.LocalAddr()
}
