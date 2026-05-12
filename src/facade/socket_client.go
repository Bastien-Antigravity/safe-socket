package facade

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
	"github.com/Bastien-Antigravity/safe-socket/src/transports"
	"sync"
)

// SocketClient implements the interfaces.Socket interface for Client-side operations.
// It handles establishing connections (Open), sending/receiving data, and protocol handshakes.
// Server-side methods (Listen, Accept) will return errors.
type SocketClient struct {
	Profile   interfaces.SocketProfile
	Config    models.SocketConfig
	transport interfaces.TransportConnection
	Logger    interfaces.Logger
	mu        sync.RWMutex
}

// -----------------------------------------------------------------------------

func NewSocketClient(p interfaces.SocketProfile, c models.SocketConfig) *SocketClient {
	return &SocketClient{
		Profile: p,
		Config:  c,
	}
}

// -----------------------------------------------------------------------------

// Open establishes the connection using the configured transport and protocol.
// If MaxRetries > 0, it will attempt reconnection on failure.
func (c *SocketClient) Open() error {
	c.mu.Lock()
	if c.transport != nil {
		c.mu.Unlock()
		return errors.New("socket already open")
	}
	c.mu.Unlock()

	retries := 0
	for {
		err := c.attemptOpen()
		if err == nil {
			return nil
		}

		// Check if we should retry
		if c.Config.MaxRetries == 0 || (c.Config.MaxRetries > 0 && retries >= c.Config.MaxRetries) {
			return fmt.Errorf("failed to open socket after %d attempts: %w", retries+1, err)
		}

		retries++
		if c.Logger != nil {
			c.Logger.Warning(fmt.Sprintf("Socket open failed: %v. Retrying in %v (Attempt %d/%d)...", err, c.Config.RetryInterval, retries, c.Config.MaxRetries))
		}

		time.Sleep(c.Config.RetryInterval)
	}
}

func (c *SocketClient) attemptOpen() error {
	// 1. Create Transport & Connect
	var conn interfaces.TransportConnection
	var err error

	// Connect Timeout (Dial)
	connectTimeout := time.Duration(c.Profile.GetConnectTimeout()) * time.Millisecond
	if connectTimeout == 0 {
		connectTimeout = 5 * time.Second
	}

	// Idle Timeout (Read/Write)
	// If Config.Deadline is set (even to 0), we use it as the Idle Timeout.
	idleTimeout := connectTimeout
	if c.Config.Deadline >= 0 {
		idleTimeout = c.Config.Deadline
	}

	switch c.Profile.GetTransport() {
	case interfaces.TransportTLS:
		conn, err = transports.ConnectTLS(c.Profile.GetAddress(), idleTimeout, c.Config.CertFile, c.Config.KeyFile, c.Config.CAFile, c.Config.ServerName, c.Config.InsecureSkipVerify)
	case interfaces.TransportFramedTCP:
		conn, err = transports.Connect(c.Profile.GetAddress(), idleTimeout)
	case interfaces.TransportShm:
		conn, err = transports.ConnectShm(c.Profile.GetName(), idleTimeout)
	case interfaces.TransportUDP:
		conn, err = transports.ConnectUDP(c.Profile.GetAddress(), idleTimeout)
	default:
		return errors.New("unsupported transport type")
	}

	if err != nil {
		return err
	}

	// 1b. Apply Reliability Layer if requested (UDP only)
	if c.Config.Reliable && c.Profile.GetTransport() == interfaces.TransportUDP {
		conn = NewReliableConnection(conn)
	}

	// 2. Encapsulation / Handshake Logic
	if c.Profile.GetTransport() == interfaces.TransportUDP &&
		c.Profile.GetProtocol() == interfaces.ProtocolHello {
		conn = NewEnvelopedConnection(conn, c.Profile, c.Config)
	} else if c.Profile.GetProtocol() != "" && c.Profile.GetProtocol() != interfaces.ProtocolNone {
		proto := protocols.NewHelloProtocol()
		if err := proto.Initiate(conn, c.Profile, c.Config); err != nil {
			_ = conn.Close()
			return err
		}
	}

	// 3. Heartbeat Optimization & Safety Ratio
	heartbeatInterval := c.Config.HeartbeatInterval
	if heartbeatInterval == 0 {
		heartbeatInterval = time.Duration(float64(idleTimeout) / 2.5)
	} else if idleTimeout > 0 && float64(heartbeatInterval)*2.5 > float64(idleTimeout) {
		// If user provided an unsafe heartbeat (too close to deadline), adjust it
		newHeartbeat := time.Duration(float64(idleTimeout) / 2.5)
		if c.Logger != nil {
			c.Logger.Warning(fmt.Sprintf("User HeartbeatInterval (%v) is too close to IdleTimeout (%v). Adjusting to safety ratio: %v",
				heartbeatInterval, idleTimeout, newHeartbeat))
		}
		heartbeatInterval = newHeartbeat
	}

	// Threshold Check (Network: 300ms, Local: 150ms, SHM: 50ms)
	threshold := 300 * time.Millisecond
	addr := c.Profile.GetAddress()
	isLocal := strings.Contains(addr, "127.0.0.1") || strings.Contains(addr, "localhost")
	isShm := c.Profile.GetTransport() == interfaces.TransportShm

	if isShm {
		threshold = 50 * time.Millisecond
	} else if isLocal {
		threshold = 150 * time.Millisecond
	}

	if idleTimeout > 0 && idleTimeout < threshold {
		heartbeatInterval = 0 // Disabled
	}

	c.mu.Lock()
	c.transport = NewHeartbeatConnection(conn, heartbeatInterval)
	c.mu.Unlock()
	return nil
}

// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------

// Send writes the raw data to the transport.
func (c *SocketClient) Send(data []byte) error {
	c.mu.RLock()
	tr := c.transport
	c.mu.RUnlock()

	if tr == nil {
		return errors.New("socket not open")
	}
	_, err := tr.Write(data)
	return err
}

// Write implements the io.Writer interface in logger.
func (c *SocketClient) Write(data []byte) (int, error) {
	c.mu.RLock()
	tr := c.transport
	c.mu.RUnlock()

	if tr == nil {
		return 0, errors.New("socket not open")
	}
	n, err := tr.Write(data)
	return n, err
}

// -----------------------------------------------------------------------------

// Receive reads from the transport into a newly allocated buffer.
// It returns the data read and any error encountered.
func (c *SocketClient) Receive() ([]byte, error) {
	c.mu.RLock()
	tr := c.transport
	c.mu.RUnlock()

	if tr == nil {
		return nil, errors.New("socket not open")
	}
	return tr.ReadMessage()
}

// Read reads from the transport into the provided buffer (io.Reader compliance).
func (c *SocketClient) Read(p []byte) (int, error) {
	c.mu.RLock()
	tr := c.transport
	c.mu.RUnlock()

	if tr == nil {
		return 0, errors.New("socket not open")
	}
	return tr.Read(p)
}

// -----------------------------------------------------------------------------

func (c *SocketClient) Close() error {
	c.mu.Lock()
	tr := c.transport
	c.transport = nil
	c.mu.Unlock()

	if tr != nil {
		return tr.Close()
	}
	return nil
}

// -----------------------------------------------------------------------------

// SetDeadline sets the read and write deadlines associated with the connection.
func (c *SocketClient) SetDeadline(t time.Time) error {
	c.mu.RLock()
	tr := c.transport
	c.mu.RUnlock()

	if tr == nil {
		return errors.New("socket not open")
	}
	return tr.SetDeadline(t)
}

// -----------------------------------------------------------------------------

// SetReadDeadline sets the deadline for future Read calls.
func (c *SocketClient) SetReadDeadline(t time.Time) error {
	c.mu.RLock()
	tr := c.transport
	c.mu.RUnlock()

	if tr == nil {
		return errors.New("socket not open")
	}
	return tr.SetReadDeadline(t)
}

// -----------------------------------------------------------------------------

// SetWriteDeadline sets the deadline for future Write calls.
func (c *SocketClient) SetWriteDeadline(t time.Time) error {
	c.mu.RLock()
	tr := c.transport
	c.mu.RUnlock()

	if tr == nil {
		return errors.New("socket not open")
	}
	return tr.SetWriteDeadline(t)
}

// -----------------------------------------------------------------------------

// SetIdleTimeout updates the internal idle timeout and refreshes the current deadline.
func (c *SocketClient) SetIdleTimeout(d time.Duration) error {
	c.mu.Lock()
	c.Config.Deadline = d
	tr := c.transport
	c.mu.Unlock()

	if tr != nil {
		return tr.SetIdleTimeout(d)
	}
	return nil
}

// -----------------------------------------------------------------------------

// Bind logger to safe-socket
func (c *SocketClient) SetLogger(logger interfaces.Logger) {
	c.Logger = logger
}

// -----------------------------------------------------------------------------
// Server Methods (Not Supported for Client)
// -----------------------------------------------------------------------------

func (c *SocketClient) Listen() error {
	return errors.New("method Listen not supported for Client socket")
}

func (c *SocketClient) Accept() (interfaces.TransportConnection, error) {
	return nil, errors.New("method Accept not supported for Client socket")
}
