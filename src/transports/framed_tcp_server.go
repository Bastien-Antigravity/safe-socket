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

// FramedTCPListener implements interfaces.TransportListener.
type FramedTCPListener struct {
	Listener net.Listener
	Timeout  time.Duration
}

// -----------------------------------------------------------------------------

// Accept waits for and returns the next connection to the listener.
func (l *FramedTCPListener) Accept() (interfaces.TransportConnection, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	// Wrap the raw net.Conn in our FramedTCPSocket
	socket := NewFramedTCPSocket(conn, l.Timeout)

	// Apply TCP Optimizations
	_ = socket.SetKeepAlive(30 * time.Second)
	_ = socket.SetNoDelay(true)
	_ = socket.SetReadBuffer(4 * 1024 * 1024)
	_ = socket.SetWriteBuffer(4 * 1024 * 1024)

	return socket, nil
}

// -----------------------------------------------------------------------------

// Close closes the listener.
func (l *FramedTCPListener) Close() error {
	return l.Listener.Close()
}

// -----------------------------------------------------------------------------

// Addr returns the listener's network address.
func (l *FramedTCPListener) Addr() net.Addr {
	return l.Listener.Addr()
}

// -----------------------------------------------------------------------------

// Listen creates a new FramedTCPListener.
func Listen(address string, timeout time.Duration) (interfaces.TransportListener, error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &FramedTCPListener{
		Listener: ln,
		Timeout:  timeout,
	}, nil
}

// ListenTLS creates a new TLS-enabled FramedTCPListener.
func ListenTLS(address string, timeout time.Duration, certFile, keyFile, caFile string) (interfaces.TransportListener, error) {
	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("TLS listener requires certFile and keyFile")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Load CA if provided (for mTLS)
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	ln, err := tls.Listen("tcp", address, tlsConfig)
	if err != nil {
		return nil, err
	}

	return &FramedTCPListener{
		Listener: ln,
		Timeout:  timeout,
	}, nil
}
