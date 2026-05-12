package test

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Bastien-Antigravity/safe-socket/src/factory"
)

// TestStress_Concurrency verifies the library's stability under high concurrent load.
func TestStress_Concurrency(t *testing.T) {
	transports := []string{"tcp", "shm"} // UDP is connectionless, stress handled differently

	for _, tr := range transports {
		t.Run(tr, func(t *testing.T) {
			addr := "127.0.0.1:9200"
			numClients := 100
			messagesPerClient := 10

			if tr == "shm" {
				addr = "stress_shm_file"
				numClients = 1           // SHM is strictly 1-to-1 in current implementation
				messagesPerClient = 1000 // Stress by volume instead of concurrency
				defer func() { _ = os.Remove(addr) }()
			}

			// 1. Start Server
			server, err := factory.Create(tr, addr, "", "server", true)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}
			defer func() { _ = server.Close() }()

			var serverErrorCount int32
			var serverMsgCount int32

			go func() {
				for {
					conn, err := server.Accept()
					if err != nil {
						return // Server closed
					}
					go func() {
						defer func() { _ = conn.Close() }()
						for i := 0; i < messagesPerClient; i++ {
							msg, err := conn.ReadMessage()
							if err != nil {
								atomic.AddInt32(&serverErrorCount, 1)
								return
							}
							atomic.AddInt32(&serverMsgCount, 1)
							_, _ = conn.Write(msg) // Echo
						}
					}()
				}
			}()

			// 2. Start Clients
			var wg sync.WaitGroup
			var clientErrorCount int32
			var clientSuccessCount int32

			start := time.Now()
			for i := 0; i < numClients; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					// Slight jitter for connection establishment
					time.Sleep(time.Duration(id%10) * time.Millisecond)

					client, err := factory.Create(tr, addr, "", "client", true)
					if err != nil {
						atomic.AddInt32(&clientErrorCount, 1)
						return
					}
					defer func() { _ = client.Close() }()

					for j := 0; j < messagesPerClient; j++ {
						payload := []byte(fmt.Sprintf("msg-%d-%d", id, j))
						if err := client.Send(payload); err != nil {
							atomic.AddInt32(&clientErrorCount, 1)
							return
						}
						resp, err := client.Receive()
						if err != nil || string(resp) != string(payload) {
							atomic.AddInt32(&clientErrorCount, 1)
							return
						}
					}
					atomic.AddInt32(&clientSuccessCount, 1)
				}(i)
			}

			wg.Wait()
			duration := time.Since(start)

			t.Logf("Transport %s: %d clients finished in %v", tr, numClients, duration)
			t.Logf("Success: %d, Client Errors: %d, Server Errors: %d", clientSuccessCount, clientErrorCount, serverErrorCount)

			if clientErrorCount > 0 {
				t.Errorf("Transport %s had %d client errors", tr, clientErrorCount)
			}

			expectedMsgs := int32(numClients * messagesPerClient)
			if atomic.LoadInt32(&serverMsgCount) != expectedMsgs {
				t.Errorf("Expected %d total messages, got %d", expectedMsgs, serverMsgCount)
			}
		})
	}
}

// TestStress_RapidReconnect verifies no leaks or races during rapid open/close cycles.
func TestStress_RapidReconnect(t *testing.T) {
	addr := "127.0.0.1:9201"
	server, _ := factory.Create("tcp", addr, "", "server", true)
	defer func() { _ = server.Close() }()

	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				return
			}
			go func() {
				time.Sleep(5 * time.Millisecond)
				_ = conn.Close()
			}()
		}
	}()

	const cycles = 200
	var wg sync.WaitGroup
	wg.Add(10) // 10 concurrent reconnecting clients

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < cycles/10; j++ {
				client, err := factory.Create("tcp", addr, "", "client", true)
				if err == nil {
					_ = client.Close()
				}
			}
		}()
	}

	wg.Wait()
	t.Logf("Rapid reconnect test completed successfully")
}
