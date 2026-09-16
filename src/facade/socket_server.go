package facade

// =============================================================================
// ESSENTIAL PROCESS:
// Implements the interfaces.Socket interface for server-side lifecycle management,
// handling transport listening, connection tracking, handshakes, and graceful draining.
//
// DATA FLOW:
// 1. Input: Incoming network transport connections from TransportListener.
// 2. Logic: Enforces handshakes, tracks active connections, monitors idle timeouts,
//    and coordinates timeout-guarded synchronous shutdown.
// 3. Output: Accepted TransportConnection streams wrapped with protocol facades.
//
// KEY PARAMETERS:
// - Profile: Socket profile specifying transport protocol and bind address.
// - Config: Socket configuration providing deadlines and security credentials.
// =============================================================================

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"sync"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
	"github.com/Bastien-Antigravity/safe-socket/src/protocols"
	"github.com/Bastien-Antigravity/safe-socket/src/transports"
)

// -----------------------------------------------------------------------------

// SocketServer implements the interfaces.Socket interface for Server-side operations.
// It handles listening for connections (Listen) and accepting them (Accept).
// It also executes the handshake protocol (if configured) during Accept.
// Client-side methods (Open, Send, Receive) will return errors, as the Server itself
// does not send/receive data directly; the *accepted connection* does.
type SocketServer struct {
	Profile     interfaces.SocketProfile
	Config      models.SocketConfig
	listener    interfaces.TransportListener
	Logger      interfaces.Logger
	wg          sync.WaitGroup
	mu          sync.RWMutex
	activeConns map[*trackingConnection]struct{}
	connMu      sync.Mutex
}

// -----------------------------------------------------------------------------

// NewSocketServer creates a new instance of SocketServer.
func NewSocketServer(p interfaces.SocketProfile, config models.SocketConfig) *SocketServer {
	return &SocketServer{
		Profile:     p,
		Config:      config,
		activeConns: make(map[*trackingConnection]struct{}),
	}
}

// -----------------------------------------------------------------------------

// Listen starts listening on the address specified by the profile.
func (s *SocketServer) Listen() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return errors.New("server already listening")
	}

	var ln interfaces.TransportListener
	var err error

	timeout := time.Duration(s.Profile.GetConnectTimeout()) * time.Millisecond

	switch s.Profile.GetTransport() {
	case interfaces.TransportTLS:
		ln, err = transports.ListenTLS(s.Profile.GetAddress(), timeout, s.Config.CertFile, s.Config.KeyFile, s.Config.CAFile)
	case interfaces.TransportFramedTCP:
		ln, err = transports.Listen(s.Profile.GetAddress(), timeout)
	case interfaces.TransportUDP:
		ln, err = transports.ListenUDP(s.Profile.GetAddress(), timeout)
	case interfaces.TransportShm:
		ln, err = transports.ListenShm(s.Profile.GetAddress(), timeout)
	default:
		return errors.New("unsupported transport type for listening")
	}

	if err != nil {
		return err
	}

	s.listener = ln
	return nil
}

// -----------------------------------------------------------------------------

// Accept accepts a new connection and performs the handshake if defined.
func (s *SocketServer) Accept() (interfaces.TransportConnection, error) {
	s.mu.RLock()
	ln := s.listener
	s.mu.RUnlock()

	if ln == nil {
		return nil, errors.New("server not listening")
	}

	// 1. Accept raw transport connection
	conn, err := ln.Accept()
	if err != nil {
		return nil, err
	}

	// 1a. Track connection for synchronous shutdown
	s.wg.Add(1)
	tracker := &trackingConnection{
		TransportConnection: conn,
	}
	tracker.onClose = func() {
		s.connMu.Lock()
		delete(s.activeConns, tracker)
		s.connMu.Unlock()
		s.wg.Done()
	}
	s.connMu.Lock()
	if s.activeConns == nil {
		s.activeConns = make(map[*trackingConnection]struct{})
	}
	s.activeConns[tracker] = struct{}{}
	s.connMu.Unlock()

	conn = tracker

	// 1b. Apply Server Config Deadline (Idle Timeout)
	// If s.Config.Deadline is set (even to 0), we use it as the Idle Timeout.
	idleTimeout := time.Duration(s.Profile.GetConnectTimeout()) * time.Millisecond
	if s.Config.Deadline >= 0 {
		idleTimeout = s.Config.Deadline
	}
	_ = conn.SetIdleTimeout(idleTimeout)

	// 1c. Apply Reliability Layer if requested (UDP only)
	if s.Config.Reliable && s.Profile.GetTransport() == interfaces.TransportUDP {
		conn = NewReliableConnection(conn)
	}

	// 2. Encapsulation / Handshake Logic
	// Case A: UDP + Hello (Stateless Envelope)
	if s.Profile.GetTransport() == interfaces.TransportUDP &&
		s.Profile.GetProtocol() == interfaces.ProtocolHello {

		// Wrap connection to handle Per-Packet Decapsulation
		config := models.SocketConfig{} // Server doesn't usually use config for receiving, but wrapper needs it struct
		conn = NewEnvelopedConnection(conn, s.Profile, config)

	} else if s.Profile.GetProtocol() != "" && s.Profile.GetProtocol() != interfaces.ProtocolNone {
		// Case B: Connection-Oriented (TCP) + Hello
		// Perform Standard Handshake (Wait for Client to send Hello)
		proto := protocols.NewHelloProtocol()

		// Note: The handshake itself will respect the Deadline set in 1b because it uses Read/Write on the conn.
		helloMsg, err := proto.WaitInitiation(conn)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}

		// Wrap with identity
		conn = NewHandshakeConnection(conn, helloMsg)
	}

	// 3. Heartbeat Optimization & Safety Ratio
	heartbeatInterval := s.Config.HeartbeatInterval
	if heartbeatInterval == 0 {
		heartbeatInterval = time.Duration(float64(idleTimeout) / 2.5)
	} else if idleTimeout > 0 && float64(heartbeatInterval)*2.5 > float64(idleTimeout) {
		// If user provided an unsafe heartbeat (too close to deadline), adjust it
		newHeartbeat := time.Duration(float64(idleTimeout) / 2.5)
		if s.Logger != nil {
			s.Logger.Warning(fmt.Sprintf("User HeartbeatInterval (%v) is too close to IdleTimeout (%v). Adjusting to safety ratio: %v",
				heartbeatInterval, idleTimeout, newHeartbeat))
		}
		heartbeatInterval = newHeartbeat
	}

	// Threshold Check (Network: 300ms, Local: 150ms, SHM: 50ms)
	threshold := 300 * time.Millisecond // Default (Networking)
	addr := s.Profile.GetAddress()
	isLocal := strings.Contains(addr, "127.0.0.1") || strings.Contains(addr, "localhost")
	isShm := s.Profile.GetTransport() == interfaces.TransportShm

	transportName := "networking"
	if isShm {
		threshold = 50 * time.Millisecond
		transportName = "shared memory"
	} else if isLocal {
		threshold = 150 * time.Millisecond
		transportName = "local"
	}

	if idleTimeout > 0 && idleTimeout < threshold {
		if s.Logger != nil {
			s.Logger.Info(fmt.Sprintf("Heartbeat disabled: IdleTimeout (%v) is below the threshold for %s transport.", idleTimeout, transportName))
		}
		return NewHeartbeatConnection(conn, 0), nil
	}

	return NewHeartbeatConnection(conn, heartbeatInterval), nil
}

// -----------------------------------------------------------------------------

// GetAddr returns the listener's network address, if the server is listening.
func (s *SocketServer) GetAddr() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener == nil {
		return "", errors.New("server not listening")
	}
	return s.listener.Addr().String(), nil
}

// -----------------------------------------------------------------------------

// Close stops the server and waits for active connections to finish within Config.Deadline.
// If connections remain open after the drain timeout, they are force-closed to prevent shutdown hangs.
func (s *SocketServer) Close() error {
	s.mu.Lock()
	ln := s.listener
	s.listener = nil
	s.mu.Unlock()

	if ln == nil {
		return nil
	}

	err := ln.Close()

	waitDone := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitDone)
	}()

	drainTimeout := s.Config.Deadline
	if drainTimeout <= 0 {
		drainTimeout = 5 * time.Second
	}

	select {
	case <-waitDone:
		// Clean graceful shutdown completed within deadline
	case <-time.After(drainTimeout):
		// Drain timeout reached: force-close any remaining active connections
		s.connMu.Lock()
		conns := make([]*trackingConnection, 0, len(s.activeConns))
		for c := range s.activeConns {
			conns = append(conns, c)
		}
		s.connMu.Unlock()

		for _, c := range conns {
			_ = c.Close()
		}

		// Wait briefly for wg to drain now that connections are closed
		select {
		case <-waitDone:
		case <-time.After(1 * time.Second):
			// Failsafe
		}
	}

	return err
}

// trackingConnection wraps a TransportConnection to signal when it's closed.
type trackingConnection struct {
	interfaces.TransportConnection
	onClose func()
	once    sync.Once
}

func (c *trackingConnection) Close() error {
	var err error
	c.once.Do(func() {
		err = c.TransportConnection.Close()
		c.onClose()
	})
	return err
}

// -----------------------------------------------------------------------------

// Bind logger to safe-socket
func (s *SocketServer) SetLogger(logger interfaces.Logger) {
	s.Logger = logger
}

// -----------------------------------------------------------------------------
// Client Methods (Not Supported for Server)
// -----------------------------------------------------------------------------

func (s *SocketServer) Open() error {
	return errors.New("method Open not supported for Server socket")
}

func (s *SocketServer) Send(data []byte) error {
	return errors.New("method Send not supported for Server socket")
}

func (s *SocketServer) Write(data []byte) (int, error) {
	return 0, errors.New("method Write not supported for Server socket")
}

func (s *SocketServer) Receive() ([]byte, error) {
	return nil, errors.New("method Receive not supported for Server socket")
}

func (s *SocketServer) Read(p []byte) (int, error) {
	return 0, errors.New("method Read not supported for Server socket")
}

func (s *SocketServer) SetDeadline(t time.Time) error {
	return errors.New("method SetDeadline not supported for Server listener (use Config.Deadline for accepted conns)")
}

func (s *SocketServer) SetReadDeadline(t time.Time) error {
	return errors.New("method SetReadDeadline not supported for Server listener")
}

func (s *SocketServer) SetWriteDeadline(t time.Time) error {
	return errors.New("method SetWriteDeadline not supported for Server listener")
}

// SetIdleTimeout updates the internal idle timeout for newly accepted connections.
func (s *SocketServer) SetIdleTimeout(d time.Duration) error {
	s.Config.Deadline = d
	return nil
}
