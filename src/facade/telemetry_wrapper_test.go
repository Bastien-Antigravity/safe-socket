package facade

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

type mockTransportConn struct {
	net.Conn
	lastWritten []byte
	readPayload []byte
}

func (m *mockTransportConn) Write(p []byte) (int, error) {
	m.lastWritten = append([]byte(nil), p...)
	return len(p), nil
}

func (m *mockTransportConn) Read(p []byte) (int, error) {
	n := copy(p, m.readPayload)
	return n, nil
}

func (m *mockTransportConn) ReadMessage() ([]byte, error) {
	return m.readPayload, nil
}

func (m *mockTransportConn) Close() error {
	return nil
}

func (m *mockTransportConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54000}
}

func (m *mockTransportConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9090}
}

func (m *mockTransportConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockTransportConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockTransportConn) SetWriteDeadline(t time.Time) error { return nil }
func (m *mockTransportConn) SetIdleTimeout(d time.Duration) error { return nil }

var _ interfaces.TransportConnection = (*mockTransportConn)(nil)

func TestTelemetryWrapperAtomicSwitchAndSegregation(t *testing.T) {
	var mu sync.Mutex
	var events []TelemetryEvent

	SetTelemetryHook(func(ev TelemetryEvent) {
		mu.Lock()
		events = append(events, ev)
		mu.Unlock()
	})
	defer func() {
		SetTelemetryEnabled(false)
		SetTelemetryHook(nil)
	}()

	// 1. Disabled by default: zero events should be emitted
	SetTelemetryEnabled(false)
	if IsTelemetryEnabled() {
		t.Fatalf("expected telemetry to be disabled")
	}

	mock := &mockTransportConn{readPayload: []byte("hello-frame")}
	tc := NewTelemetryConnection(mock, "client", "tcp", "tcp-hello", "log_server", "", "")

	if _, err := tc.Write([]byte("outbound-payload")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if _, err := tc.ReadMessage(); err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	mu.Lock()
	if len(events) != 0 {
		t.Fatalf("expected 0 events while disabled, got %d", len(events))
	}
	mu.Unlock()

	// 2. Enable atomically at runtime: subsequent calls on existing and new connections emit events
	SetTelemetryEnabled(true)
	if !IsTelemetryEnabled() {
		t.Fatalf("expected telemetry to be enabled")
	}

	// Write data frame
	if _, err := tc.Write([]byte("live-log-entry")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	// Write heartbeat (0-byte frame)
	if _, err := tc.Write([]byte{}); err != nil {
		t.Fatalf("Heartbeat Write failed: %v", err)
	}
	// Read message frame
	if _, err := tc.ReadMessage(); err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	// Close connection
	if err := tc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	FlushTelemetry(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 4 {
		t.Fatalf("expected 4 telemetry events (write frame, write heartbeat, read frame, disconnect), got %d", len(events))
	}

	if events[0].EventType != TelemetryEventWrite || events[0].Channel != "safesocket:frame" || events[0].Bytes != 14 {
		t.Errorf("unexpected event[0]: %+v", events[0])
	}
	if events[1].EventType != TelemetryEventWrite || events[1].Channel != "safesocket:heartbeat" || events[1].Bytes != 0 {
		t.Errorf("unexpected event[1] (heartbeat): %+v", events[1])
	}
	if events[2].EventType != TelemetryEventRead || events[2].Channel != "safesocket:frame" || events[2].Bytes != 11 {
		t.Errorf("unexpected event[2] (read): %+v", events[2])
	}
	if events[3].EventType != TelemetryEventDisconnect || events[3].Channel != "safesocket:lifecycle" {
		t.Errorf("unexpected event[3] (disconnect): %+v", events[3])
	}

	bSent, bRecv, fSent, fRecv := tc.Stats()
	if bSent != 14 || bRecv != 11 || fSent != 2 || fRecv != 1 {
		t.Errorf("unexpected atomic stats: sent=%d/%d, recv=%d/%d", bSent, fSent, bRecv, fRecv)
	}
}

func TestTelemetryNonBlockingSlowHook(t *testing.T) {
	hookEntered := make(chan struct{})
	SetTelemetryHook(func(ev TelemetryEvent) {
		select {
		case hookEntered <- struct{}{}:
		default:
		}
		time.Sleep(100 * time.Millisecond) // Slow consumer
	})
	SetTelemetryEnabled(true)
	defer func() {
		SetTelemetryEnabled(false)
		SetTelemetryHook(nil)
	}()

	mock := &mockTransportConn{readPayload: []byte("payload")}
	tc := NewTelemetryConnection(mock, "client", "tcp", "tcp-hello", "peer", "", "")

	start := time.Now()
	// Write must return almost instantly (< 10ms) without waiting for the 100ms slow hook
	_, err := tc.Write([]byte("quick-write"))
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("tc.Write failed: %v", err)
	}
	if duration > 20*time.Millisecond {
		t.Errorf("expected non-blocking Write to return in <20ms, took %v", duration)
	}

	select {
	case <-hookEntered:
	case <-time.After(200 * time.Millisecond):
		t.Errorf("hook was not executed asynchronously")
	}
}

func TestTelemetryPanicResilience(t *testing.T) {
	SetTelemetryHook(func(ev TelemetryEvent) {
		panic("deliberate consumer callback panic")
	})
	SetTelemetryEnabled(true)
	defer func() {
		SetTelemetryEnabled(false)
		SetTelemetryHook(nil)
		FlushTelemetry(50 * time.Millisecond)
	}()

	mock := &mockTransportConn{readPayload: []byte("payload")}
	tc := NewTelemetryConnection(mock, "client", "tcp", "tcp-hello", "peer", "", "")

	// Write and ReadMessage must not panic
	n, err := tc.Write([]byte("test-payload"))
	if err != nil || n != 12 {
		t.Errorf("unexpected Write result: n=%d err=%v", n, err)
	}

	FlushTelemetry(50 * time.Millisecond)
}

func TestZeroOverheadWhenDisabled(t *testing.T) {
	SetTelemetryEnabled(false)
	SetTelemetryHook(nil)
	FlushTelemetry(50 * time.Millisecond)
	defer SetTelemetryEnabled(false)

	mock := &mockTransportConn{}
	wrapped := WrapConnectionWithTelemetry(mock, "client", "tcp", "tcp-hello", "peer", "", "")

	// When disabled, WrapConnectionWithTelemetry must return the raw underlying conn directly
	if wrapped != mock {
		t.Errorf("expected wrapped to be raw mock connection %p, got %p", mock, wrapped)
	}
	if _, ok := wrapped.(*TelemetryConnection); ok {
		t.Errorf("expected wrapped NOT to be *TelemetryConnection when disabled")
	}

	// When enabled, it must wrap in *TelemetryConnection
	SetTelemetryEnabled(true)
	wrappedEnabled := WrapConnectionWithTelemetry(mock, "client", "tcp", "tcp-hello", "peer", "", "")
	if _, ok := wrappedEnabled.(*TelemetryConnection); !ok {
		t.Errorf("expected wrappedEnabled to be *TelemetryConnection when enabled")
	}
}

func TestSegregatedChannelExtraction(t *testing.T) {
	SetTelemetryEnabled(false)
	SetTelemetryHook(nil)
	FlushTelemetry(50 * time.Millisecond)

	var mu sync.Mutex
	var channels []string

	SetTelemetryHook(func(ev TelemetryEvent) {
		mu.Lock()
		channels = append(channels, ev.Channel)
		mu.Unlock()
	})
	SetTelemetryEnabled(true)
	defer func() {
		SetTelemetryEnabled(false)
		SetTelemetryHook(nil)
		FlushTelemetry(50 * time.Millisecond)
	}()

	mock := &mockTransportConn{}
	tc := NewTelemetryConnection(mock, "client", "tcp", "tcp-hello", "peer", "", "")

	// Write empty frame -> heartbeat
	_, _ = tc.Write([]byte{})
	// Write SYNC_CONFIG -> sync_config
	_, _ = tc.Write([]byte("SYNC_CONFIG:v1"))
	// Write LOG_BATCH -> log_batch
	_, _ = tc.Write([]byte("LOG_BATCH:entry"))
	// Write standard frame -> frame
	_, _ = tc.Write([]byte("OTHER_DATA"))

	FlushTelemetry(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	expectedChannels := []string{
		"safesocket:lifecycle", // connect from NewTelemetryConnection
		"safesocket:heartbeat",
		"safesocket:sync_config",
		"safesocket:log_batch",
		"safesocket:frame",
	}

	if len(channels) != len(expectedChannels) {
		t.Fatalf("expected %d channels, got %d: %v", len(expectedChannels), len(channels), channels)
	}
	for i, exp := range expectedChannels {
		if channels[i] != exp {
			t.Errorf("channel[%d]: expected %q, got %q", i, exp, channels[i])
		}
	}
}
