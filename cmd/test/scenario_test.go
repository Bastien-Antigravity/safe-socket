package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/factory"
	"github.com/Bastien-Antigravity/safe-socket/src/models"
)

// TestScenario_CustomParameters demonstrates how to parameterize the socket
// for custom scenario testing (latency, timeouts, heartbeats).
func TestScenario_CustomParameters(t *testing.T) {
	addr := "127.0.0.1:9999"

	// --- CUSTOM PARAMETERS ---
	// You can customize these values to simulate different network scenarios.
	customConfig := models.SocketConfig{
		PublicIP:          "1.2.3.4",
		HandshakeTimeout:  150 * time.Millisecond, // Tight handshake for high-speed infra
		Deadline:          100 * time.Millisecond, // Very tight data deadline
		HeartbeatInterval: 500 * time.Millisecond, // Rapid heartbeats
	}

	fmt.Printf("Starting scenario test with parameters:\n")
	fmt.Printf(" - HandshakeTimeout: %v\n", customConfig.HandshakeTimeout)
	fmt.Printf(" - Data Deadline:    %v\n", customConfig.Deadline)
	fmt.Printf(" - Heartbeat:        %v\n", customConfig.HeartbeatInterval)

	// 1. Create Server with custom parameters
	server, err := factory.CreateWithConfig("tcp-hello:scenario-server", addr, customConfig, "server", true)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// 2. Create Client with custom parameters
	client, err := factory.CreateWithConfig("tcp-hello:scenario-client", addr, customConfig, "client", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Open the client (performs handshake with custom timing)
	start := time.Now()
	if err := client.Open(); err != nil {
		t.Fatalf("Handshake failed with custom parameters: %v", err)
	}
	fmt.Printf("Handshake completed in %v\n", time.Since(start))
	defer client.Close()

	// 3. Verify Deadline enforcement
	// Server waits, client does nothing, should timeout in ~100ms
	errChan := make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			errChan <- err
			return
		}
		defer conn.Close()

		buf := make([]byte, 10)
		_, err = conn.Read(buf)
		errChan <- err // Expecting i/o timeout
	}()

	select {
	case err := <-errChan:
		if err != nil && (time.Since(start) > 50*time.Millisecond) {
			fmt.Printf("Successfully triggered tight deadline error: %v\n", err)
		} else if err == nil {
			t.Errorf("Expected timeout but operation succeeded")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Scenario test timed out without triggering deadline")
	}
}
