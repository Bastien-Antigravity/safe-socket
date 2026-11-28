package facade

// =============================================================================
// ESSENTIAL PROCESS:
// Implements the interfaces.ClientSocket and interfaces.Socket contracts for
// client-side transport management, TLS/TCP/UDP/SHM setup, protocol handshakes,
// and thread-safe data sending and receiving.
//
// DATA FLOW:
// 1. Input: Target endpoint address, SocketProfile, and SocketConfig.
// 2. Logic: Dials transport once, runs protocol handshake, decorates with heartbeat
//    and reliability wrappers, and manages I/O. (Retries belong in conn_manager.)
// 3. Output: Transmits and receives framed payloads across established streams.
//
// KEY PARAMETERS:
// - Profile: Socket profile configuring transport type and handshake protocol.
// - Config: Client configuration specifying timeouts, deadlines, and certificates.
// =============================================================================

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
	"github.com/Bastien-Antigravity/safe-socket/src/transports"
	"github.com/Bastien-Antigravity/safe-socket/src/utils"
)

// SocketClient implements interfaces.ClientSocket and interfaces.Socket for Client-side operations.
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
		Logger:  interfaces.EnsureSafeLogger(nil),
	}
}

// -----------------------------------------------------------------------------

// Open establishes a single connection attempt using the configured transport and protocol.
// Backoff and reconnection orchestration are delegated to higher layers (e.g. conn_manager).
func (c *SocketClient) Open() error {
	c.mu.Lock()
	if c.transport != nil {
		c.mu.Unlock()
		return errors.New("socket already open")
	}
	c.mu.Unlock()

	return c.attemptOpen()
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
	idleTimeout := connectTimeout
	if c.Config.DisableTimeout || c.Config.Deadline < 0 {
		idleTimeout = 0
	} else if c.Config.Deadline > 0 {
		idleTimeout = c.Config.Deadline
	}

	switch c.Profile.GetTransport() {
	case interfaces.TransportTLS:
		conn, err = transports.ConnectTLS(c.Profile.GetAddress(), connectTimeout, c.Config.CertFile, c.Config.KeyFile, c.Config.CAFile, c.Config.ServerName, c.Config.InsecureSkipVerify)
	case interfaces.TransportFramedTCP:
		conn, err = transports.Connect(c.Profile.GetAddress(), connectTimeout)
	case interfaces.TransportShm:
		conn, err = transports.ConnectShm(c.Profile.GetAddress(), connectTimeout)
	case interfaces.TransportUDP:
		conn, err = transports.ConnectUDP(c.Profile.GetAddress(), connectTimeout)
	default:
		return errors.New("unsupported transport type")
	}

	if err != nil {
		return err
	}

	// Apply runtime idle timeout after dial (preserving connectTimeout for the dial itself)
	_ = conn.SetIdleTimeout(idleTimeout)

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
	if heartbeatInterval < 0 || idleTimeout == 0 {
		heartbeatInterval = 0
	} else if heartbeatInterval == 0 {
		heartbeatInterval = time.Duration(float64(idleTimeout) / 2.5)

		// Threshold Check (Network: 300ms, Local: 150ms, SHM: 50ms)
		threshold := 300 * time.Millisecond
		addr := c.Profile.GetAddress()
		isLocal := utils.GetMachineDetector().IsLocalAddress(addr) || strings.Contains(addr, "localhost")
		isShm := c.Profile.GetTransport() == interfaces.TransportShm

		if isShm {
			threshold = 50 * time.Millisecond
		} else if isLocal {
			threshold = 150 * time.Millisecond
		}

		if (idleTimeout > 0 && idleTimeout < threshold) || c.Profile.GetTransport() == interfaces.TransportUDP {
			heartbeatInterval = 0 // Disabled for auto-negotiation
		}
	} else if idleTimeout > 0 && heartbeatInterval < idleTimeout && float64(heartbeatInterval)*2.5 > float64(idleTimeout) {
		// If user provided an unsafe heartbeat (too close to deadline), adjust it
		newHeartbeat := time.Duration(float64(idleTimeout) / 2.5)
		c.Logger.Warning(fmt.Sprintf("User HeartbeatInterval (%v) is too close to IdleTimeout (%v). Adjusting to safety ratio: %v",
			heartbeatInterval, idleTimeout, newHeartbeat))
		heartbeatInterval = newHeartbeat
	}

	hbConn := NewHeartbeatConnection(conn, heartbeatInterval)
	profileName := ""
	if c.Profile != nil {
		profileName = c.Profile.GetName()
	}
	trackedConn := WrapConnectionWithTelemetry(
		hbConn,
		"client",
		string(c.Profile.GetTransport()),
		profileName,
		"",
		"",
		c.Profile.GetAddress(),
	)

	c.mu.Lock()
	c.transport = trackedConn
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

// ReadMessage aliases Receive to satisfy interfaces.ClientSocket and mirror TransportConnection.
func (c *SocketClient) ReadMessage() ([]byte, error) {
	return c.Receive()
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
	c.Config.DisableTimeout = d <= 0
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
	c.Logger = interfaces.EnsureSafeLogger(logger)
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
