package facade

// =============================================================================
// ESSENTIAL PROCESS:
// Provides a distinct, lock-free atomic switchable telemetry wrapper for
// safe-socket TransportConnections to track real-time inter-service connections
// and segregated frame channels (data frames vs heartbeats) with zero overhead
// when disabled.
//
// DATA FLOW:
// 1. Input: Underlying TransportConnection, peer identity, and profile metadata.
// 2. Logic: Performs an atomic.Bool load (<1ns, lock-free) on Read/Write/Close.
//    When enabled, updates atomic counters and invokes the lock-free atomic hook.
// 3. Output: Real-time TelemetryEvent stream for ecosystem topology tracking.
//
// KEY PARAMETERS:
// - telemetryEnabled: Lock-free atomic.Bool toggle switch.
// - telemetryHook: Lock-free atomic.Pointer callback for telemetry consumers.
// =============================================================================

import (
	"sync/atomic"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// TelemetryEventType categorizes the lifecycle or I/O event on a safe-socket connection.
type TelemetryEventType string

const (
	TelemetryEventConnect    TelemetryEventType = "connect"
	TelemetryEventAccept     TelemetryEventType = "accept"
	TelemetryEventRead       TelemetryEventType = "read"
	TelemetryEventWrite      TelemetryEventType = "write"
	TelemetryEventDisconnect TelemetryEventType = "disconnect"
)

// TelemetryEvent represents a single observed connection or segregated frame event.
type TelemetryEvent struct {
	EventType   TelemetryEventType
	Role        string // "client" or "server"
	Transport   string // "tcp", "tls", "udp", "shm"
	ProfileName string // Profile identifier
	PeerName    string // Authenticated peer name from HelloMsg (if available)
	LocalAddr   string
	RemoteAddr  string
	Channel     string // Segregated channel e.g. "safesocket:frame", "safesocket:heartbeat"
	Bytes       int
	HasError    bool
	Timestamp   time.Time
}

// TelemetryHookFunc is the non-blocking callback invoked when telemetry is enabled.
type TelemetryHookFunc func(ev TelemetryEvent)

var (
	// telemetryEnabled is the lock-free atomic switch controlling wrapper tracking.
	telemetryEnabled atomic.Bool

	// telemetryHook stores the registered callback without any mutex locks.
	telemetryHook atomic.Pointer[TelemetryHookFunc]
)

// -----------------------------------------------------------------------------

// SetTelemetryEnabled atomically enables or disables real-time connection tracking.
func SetTelemetryEnabled(enabled bool) {
	telemetryEnabled.Store(enabled)
}

// IsTelemetryEnabled atomically returns whether real-time connection tracking is active.
func IsTelemetryEnabled() bool {
	return telemetryEnabled.Load()
}

// SetTelemetryHook atomically registers or clears the telemetry callback hook.
func SetTelemetryHook(fn TelemetryHookFunc) {
	if fn == nil {
		telemetryHook.Store(nil)
		return
	}
	telemetryHook.Store(&fn)
}

func emitTelemetryEvent(ev TelemetryEvent) {
	if fnPtr := telemetryHook.Load(); fnPtr != nil && *fnPtr != nil {
		(*fnPtr)(ev)
	}
}

// -----------------------------------------------------------------------------

// TelemetryConnection decorates a TransportConnection with lock-free atomic telemetry.
type TelemetryConnection struct {
	interfaces.TransportConnection
	role        string
	transport   string
	profileName string
	peerName    string
	localAddr   string
	remoteAddr  string

	bytesSent  atomic.Uint64
	bytesRecv  atomic.Uint64
	framesSent atomic.Uint64
	framesRecv atomic.Uint64
	closed     atomic.Bool
}

// Ensure TelemetryConnection implements interfaces.TransportConnection.
var _ interfaces.TransportConnection = (*TelemetryConnection)(nil)

// NewTelemetryConnection wraps a TransportConnection in a lock-free switchable telemetry decorator.
func NewTelemetryConnection(
	conn interfaces.TransportConnection,
	role string,
	transport string,
	profileName string,
	peerName string,
	localAddr string,
	remoteAddr string,
) *TelemetryConnection {
	if conn != nil {
		if localAddr == "" && conn.LocalAddr() != nil {
			localAddr = conn.LocalAddr().String()
		}
		if remoteAddr == "" && conn.RemoteAddr() != nil {
			remoteAddr = conn.RemoteAddr().String()
		}
	}

	tc := &TelemetryConnection{
		TransportConnection: conn,
		role:                role,
		transport:           transport,
		profileName:         profileName,
		peerName:            peerName,
		localAddr:           localAddr,
		remoteAddr:          remoteAddr,
	}

	if telemetryEnabled.Load() {
		evType := TelemetryEventConnect
		if role == "server" {
			evType = TelemetryEventAccept
		}
		emitTelemetryEvent(TelemetryEvent{
			EventType:   evType,
			Role:        role,
			Transport:   transport,
			ProfileName: profileName,
			PeerName:    peerName,
			LocalAddr:   localAddr,
			RemoteAddr:  remoteAddr,
			Channel:     "safesocket:lifecycle",
			Timestamp:   time.Now(),
		})
	}

	return tc
}

// Write proxies data to the underlying transport and atomically records outbound metrics if enabled.
func (tc *TelemetryConnection) Write(p []byte) (int, error) {
	n, err := tc.TransportConnection.Write(p)
	if telemetryEnabled.Load() {
		if n > 0 {
			tc.bytesSent.Add(uint64(n))
		}
		tc.framesSent.Add(1)
		channel := "safesocket:frame"
		if len(p) == 0 {
			channel = "safesocket:heartbeat"
		}
		emitTelemetryEvent(TelemetryEvent{
			EventType:   TelemetryEventWrite,
			Role:        tc.role,
			Transport:   tc.transport,
			ProfileName: tc.profileName,
			PeerName:    tc.peerName,
			LocalAddr:   tc.localAddr,
			RemoteAddr:  tc.remoteAddr,
			Channel:     channel,
			Bytes:       n,
			HasError:    err != nil,
			Timestamp:   time.Now(),
		})
	}
	return n, err
}

// Read proxies stream reads and atomically records inbound metrics if enabled.
func (tc *TelemetryConnection) Read(p []byte) (int, error) {
	n, err := tc.TransportConnection.Read(p)
	if telemetryEnabled.Load() {
		if n > 0 {
			tc.bytesRecv.Add(uint64(n))
		}
		tc.framesRecv.Add(1)
		emitTelemetryEvent(TelemetryEvent{
			EventType:   TelemetryEventRead,
			Role:        tc.role,
			Transport:   tc.transport,
			ProfileName: tc.profileName,
			PeerName:    tc.peerName,
			LocalAddr:   tc.localAddr,
			RemoteAddr:  tc.remoteAddr,
			Channel:     "safesocket:frame",
			Bytes:       n,
			HasError:    err != nil,
			Timestamp:   time.Now(),
		})
	}
	return n, err
}

// ReadMessage reads a complete frame from the underlying transport and atomically records inbound metrics if enabled.
func (tc *TelemetryConnection) ReadMessage() ([]byte, error) {
	msg, err := tc.TransportConnection.ReadMessage()
	if telemetryEnabled.Load() {
		n := len(msg)
		if n > 0 {
			tc.bytesRecv.Add(uint64(n))
		}
		tc.framesRecv.Add(1)
		channel := "safesocket:frame"
		if n == 0 && err == nil {
			channel = "safesocket:heartbeat"
		}
		emitTelemetryEvent(TelemetryEvent{
			EventType:   TelemetryEventRead,
			Role:        tc.role,
			Transport:   tc.transport,
			ProfileName: tc.profileName,
			PeerName:    tc.peerName,
			LocalAddr:   tc.localAddr,
			RemoteAddr:  tc.remoteAddr,
			Channel:     channel,
			Bytes:       n,
			HasError:    err != nil,
			Timestamp:   time.Now(),
		})
	}
	return msg, err
}

// Close closes the underlying transport and atomically emits a disconnect event once if enabled.
func (tc *TelemetryConnection) Close() error {
	err := tc.TransportConnection.Close()
	if tc.closed.CompareAndSwap(false, true) && telemetryEnabled.Load() {
		emitTelemetryEvent(TelemetryEvent{
			EventType:   TelemetryEventDisconnect,
			Role:        tc.role,
			Transport:   tc.transport,
			ProfileName: tc.profileName,
			PeerName:    tc.peerName,
			LocalAddr:   tc.localAddr,
			RemoteAddr:  tc.remoteAddr,
			Channel:     "safesocket:lifecycle",
			HasError:    err != nil,
			Timestamp:   time.Now(),
		})
	}
	return err
}

// Stats returns the current atomic counters for bytes and frames on this connection.
func (tc *TelemetryConnection) Stats() (bytesSent, bytesRecv, framesSent, framesRecv uint64) {
	return tc.bytesSent.Load(), tc.bytesRecv.Load(), tc.framesSent.Load(), tc.framesRecv.Load()
}

