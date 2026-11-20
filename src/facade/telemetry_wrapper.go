package facade

// =============================================================================
// ESSENTIAL PROCESS:
// Provides a distinct, lock-free atomic switchable telemetry wrapper for
// safe-socket TransportConnections to track real-time inter-service connections
// and segregated frame channels (data frames vs heartbeats vs control frames)
// with zero overhead when disabled and non-blocking, panic-safe dispatch.
//
// DATA FLOW:
// 1. Input: Underlying TransportConnection, peer identity, and profile metadata.
// 2. Logic: Performs an atomic.Bool load (<1ns, lock-free) on Read/Write/Close.
//    When disabled, WrapConnectionWithTelemetry returns the raw connection with
//    zero heap allocations and zero decorator hops. When enabled, dispatches
//    events asynchronously through a bounded non-blocking channel with panic
//    recovery, guaranteeing zero latency and isolation for the transport I/O path.
// 3. Output: Real-time TelemetryEvent stream for ecosystem topology tracking.
//
// KEY PARAMETERS:
// - telemetryEnabled: Lock-free atomic.Bool toggle switch.
// - telemetryHook: Lock-free atomic.Pointer callback for telemetry consumers.
// - telemetryQueue: Bounded channel for non-blocking asynchronous event delivery.
// =============================================================================

import (
	"bytes"
	"sync"
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
	Channel     string // Segregated channel e.g. "safesocket:frame", "safesocket:heartbeat", "safesocket:sync_config", "safesocket:log_batch"
	Bytes       int
	HasError    bool
	Timestamp   time.Time
}

// TelemetryHookFunc is the non-blocking callback invoked when telemetry is enabled.
type TelemetryHookFunc func(ev TelemetryEvent)

const defaultTelemetryQueueSize = 8192

var (
	// telemetryEnabled is the lock-free atomic switch controlling wrapper tracking.
	telemetryEnabled atomic.Bool

	// telemetryHook stores the registered callback without any mutex locks.
	telemetryHook atomic.Pointer[TelemetryHookFunc]

	// telemetryQueue buffers events for non-blocking asynchronous dispatch.
	telemetryQueue = make(chan TelemetryEvent, defaultTelemetryQueueSize)

	// droppedEvents atomically tracks dropped telemetry events when the queue is saturated.
	droppedEvents atomic.Uint64

	// workerOnce ensures the background worker starts exactly once.
	workerOnce sync.Once
)

func init() {
	ensureTelemetryWorker()
}

func ensureTelemetryWorker() {
	workerOnce.Do(func() {
		go telemetryWorker()
	})
}

func telemetryWorker() {
	for ev := range telemetryQueue {
		if fnPtr := telemetryHook.Load(); fnPtr != nil && *fnPtr != nil {
			safeInvokeHook(*fnPtr, ev)
		}
	}
}

func safeInvokeHook(fn TelemetryHookFunc, ev TelemetryEvent) {
	defer func() {
		_ = recover() // Catch any panic from consumer hook callback without crashing transport
	}()
	fn(ev)
}

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
		// Drain any unconsumed events in the queue when clearing hook
		for len(telemetryQueue) > 0 {
			select {
			case <-telemetryQueue:
			default:
			}
		}
		return
	}
	telemetryHook.Store(&fn)
}

func emitTelemetryEvent(ev TelemetryEvent) {
	fnPtr := telemetryHook.Load()
	if fnPtr == nil || *fnPtr == nil {
		return
	}
	select {
	case telemetryQueue <- ev:
	default:
		droppedEvents.Add(1)
	}
}

// FlushTelemetry waits for all currently queued telemetry events to be dispatched,
// or until the timeout expires. Returns true if the queue was drained within timeout.
func FlushTelemetry(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for len(telemetryQueue) > 0 {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(1 * time.Millisecond)
	}
	time.Sleep(5 * time.Millisecond)
	return true
}

// DroppedTelemetryEvents returns the total count of telemetry events dropped due to queue saturation.
func DroppedTelemetryEvents() uint64 {
	return droppedEvents.Load()
}

// -----------------------------------------------------------------------------

// detectChannel classifies payload into segregated channels without mutating wire format.
func detectChannel(payload []byte) string {
	if len(payload) == 0 {
		return "safesocket:heartbeat"
	}
	if bytes.HasPrefix(payload, []byte("SYNC_CONFIG")) {
		return "safesocket:sync_config"
	}
	if bytes.HasPrefix(payload, []byte("LOG_BATCH")) {
		return "safesocket:log_batch"
	}
	return "safesocket:frame"
}

// WrapConnectionWithTelemetry returns the raw conn directly if telemetry is disabled,
// achieving the documented zero-overhead contract with zero heap allocations.
// If telemetry is enabled, it decorates conn with a TelemetryConnection.
func WrapConnectionWithTelemetry(
	conn interfaces.TransportConnection,
	role string,
	transport string,
	profileName string,
	peerName string,
	localAddr string,
	remoteAddr string,
) interfaces.TransportConnection {
	if !telemetryEnabled.Load() {
		return conn
	}
	return NewTelemetryConnection(conn, role, transport, profileName, peerName, localAddr, remoteAddr)
}

// wrapConnectionWithTelemetry is an internal alias matching the AGENTS.md rule naming.
func wrapConnectionWithTelemetry(
	conn interfaces.TransportConnection,
	role string,
	transport string,
	profileName string,
	peerName string,
	localAddr string,
	remoteAddr string,
) interfaces.TransportConnection {
	return WrapConnectionWithTelemetry(conn, role, transport, profileName, peerName, localAddr, remoteAddr)
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
		channel := detectChannel(p)
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
		var channel string
		if n > 0 {
			channel = detectChannel(p[:n])
		} else {
			channel = "safesocket:frame"
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
		channel := detectChannel(msg)
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
