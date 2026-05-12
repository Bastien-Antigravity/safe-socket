package transports

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// -----------------------------------------------------------------------------

// Connect dialer helper for FramedTCPSocket.
func Connect(address string, timeout time.Duration) (interfaces.TransportConnection, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, err
	}
	return wrapTCP(conn, timeout), nil
}

// ConnectTLS dialer helper for TLS-wrapped FramedTCPSocket.
func ConnectTLS(address string, timeout time.Duration, certFile, keyFile, caFile, serverName string, skipVerify bool) (interfaces.TransportConnection, error) {
	tlsConfig := &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: skipVerify,
	}

	// 1. Load Client Certificate if provided
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 2. Load CA if provided (for mTLS or custom CA)
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	// 3. Dial with TLS
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		return nil, err
	}

	return wrapTCP(conn, timeout), nil
}

func wrapTCP(conn net.Conn, timeout time.Duration) interfaces.TransportConnection {
	// Note: We deliberately use the 'timeout' for both connection AND subsequent read/writes
	socket := NewFramedTCPSocket(conn, timeout)

	// Apply TCP Optimizations
	// 1. KeepAlive (detect dead peers)
	_ = socket.SetKeepAlive(30 * time.Second)

	// 2. NoDelay (Disable Nagle's algorithm for lower latency)
	_ = socket.SetNoDelay(true)

	// 3. Buffer Sizes (High throughput support)
	_ = socket.SetReadBuffer(4 * 1024 * 1024)
	_ = socket.SetWriteBuffer(4 * 1024 * 1024)

	return socket
}

