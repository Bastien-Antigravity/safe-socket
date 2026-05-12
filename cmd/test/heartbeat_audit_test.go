package test

import (
	"os"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/factory"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
)

// TestHeartbeatAudit verifies that heartbeats prevent idle timeouts across all transports.
func TestHeartbeatAudit(t *testing.T) {
	transports := []string{"tcp", "udp", "shm"}

	for _, tr := range transports {
		t.Run(tr, func(t *testing.T) {
			path := "test_heartbeat_" + tr
			addr := "127.0.0.1:9100"
			if tr == "shm" {
				addr = path
				defer func() { _ = os.Remove(path) }()
			}

			// Configuration: Tight deadline (500ms), fast heartbeats (100ms)
			config := models.SocketConfig{
				Deadline:          500 * time.Millisecond,
				HeartbeatInterval: 100 * time.Millisecond,
			}

			// 1. Start Server
			server, err := factory.CreateWithConfig(tr, addr, config, "server", true)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}
			defer func() { _ = server.Close() }()

			// 2. Start Client
			client, err := factory.CreateWithConfig(tr, addr, config, "client", true)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer func() { _ = client.Close() }()

			// 3. Server Accepts
			connChan := make(chan error, 1)
			go func() {
				conn, err := server.Accept()
				if err != nil {
					connChan <- err
					return
				}
				defer func() { _ = conn.Close() }()

				// Wait longer than Deadline (500ms)
				// If heartbeats work, this should NOT timeout.
				time.Sleep(1 * time.Second)

				// Try to read
				buf := make([]byte, 10)
				_, err = conn.Read(buf)
				connChan <- err
			}()

			// Client stays alive but sends no data.
			// Heartbeats should keep the server-side connection alive.
			time.Sleep(1200 * time.Millisecond)

			// Send data to unblock server's Read
			_ = client.Send([]byte("ping"))

			select {
			case err := <-connChan:
				if err != nil {
					t.Errorf("Transport %s failed heartbeat audit: %v", tr, err)
				} else {
					t.Logf("Transport %s passed heartbeat audit", tr)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("Transport %s audit timed out", tr)
			}
		})
	}
}
