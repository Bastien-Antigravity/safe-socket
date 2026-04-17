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
}

func NewHeartbeatConnection(conn interfaces.TransportConnection, interval time.Duration) *HeartbeatConnection {
	h := &HeartbeatConnection{
		TransportConnection: conn,
		stopHeartbeat:       make(chan struct{}),
	}
	// A duration of 0 or less disables the heartbeat pulse.
	if interval > 0 {
		go h.start(interval)
	}
	return h
}

func (h *HeartbeatConnection) start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_, err := h.TransportConnection.Write([]byte{})
			if err != nil {
				// FAIL-FAST: Close the connection if heartbeat fails.
				// This fulfills the "server problem, close parent" requirement.
				_ = h.Close()
				return
			}
		case <-h.stopHeartbeat:
			return
		}
	}
}

func (h *HeartbeatConnection) Close() error {
	h.closeOnce.Do(func() {
		close(h.stopHeartbeat)
	})
	return h.TransportConnection.Close()
}

func (h *HeartbeatConnection) SetIdleTimeout(d time.Duration) error {
	return h.TransportConnection.SetIdleTimeout(d)
}
