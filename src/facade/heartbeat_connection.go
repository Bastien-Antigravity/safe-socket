package facade

import (
	"sync"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/interfaces"
)

// HeartbeatConnection wraps a transport and periodically sends 0-length heartbeats.
type HeartbeatConnection struct {
	interfaces.TransportConnection
	stopHeartbeat chan struct{}
	closeOnce     sync.Once
	mu            sync.Mutex
	writeMu       sync.Mutex
	readMu        sync.Mutex
	interval      time.Duration
}

func NewHeartbeatConnection(conn interfaces.TransportConnection, interval time.Duration) *HeartbeatConnection {
	h := &HeartbeatConnection{
		TransportConnection: conn,
		interval:            interval,
	}
	if interval > 0 {
		h.stopHeartbeat = make(chan struct{})
		go h.start(interval, h.stopHeartbeat)
	}
	return h
}

func (h *HeartbeatConnection) start(interval time.Duration, stopChan chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			//nolint:staticcheck // QF1008: Explicit selector preferred for clarity and future-proofing
			_, err := h.Write([]byte{})
			if err != nil {
				// FAIL-FAST: Close the connection if heartbeat fails.
				_ = h.TransportConnection.Close()
				return
			}
		case <-stopChan:
			return
		}
	}
}

func (h *HeartbeatConnection) Write(p []byte) (n int, err error) {
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	return h.TransportConnection.Write(p)
}

func (h *HeartbeatConnection) Read(p []byte) (n int, err error) {
	h.readMu.Lock()
	defer h.readMu.Unlock()
	return h.TransportConnection.Read(p)
}

func (h *HeartbeatConnection) ReadMessage() ([]byte, error) {
	h.readMu.Lock()
	defer h.readMu.Unlock()
	return h.TransportConnection.ReadMessage()
}

func (h *HeartbeatConnection) Close() error {
	h.closeOnce.Do(func() {
		h.mu.Lock()
		if h.stopHeartbeat != nil {
			close(h.stopHeartbeat)
			h.stopHeartbeat = nil
		}
		h.mu.Unlock()
	})
	return h.TransportConnection.Close()
}

func (h *HeartbeatConnection) SetIdleTimeout(d time.Duration) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 1. Update Transport
	err := h.TransportConnection.SetIdleTimeout(d)
	if err != nil {
		return err
	}

	// 2. Manage Heartbeat Ticker
	// Calculate new interval (SafeSocket standard: Deadline / 2.5)
	newInterval := time.Duration(float64(d) / 2.5)

	// Stop existing ticker
	if h.stopHeartbeat != nil {
		close(h.stopHeartbeat)
		h.stopHeartbeat = nil
	}

	// Start new ticker if needed
	if newInterval > 0 {
		h.stopHeartbeat = make(chan struct{})
		go h.start(newInterval, h.stopHeartbeat)
	}

	h.interval = newInterval
	return nil
}
