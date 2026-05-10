package transports

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/edsrzf/mmap-go"
)

func TestZombieDetection(t *testing.T) {
	t.Run("Standard_Zombie_Detection", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()

		go func() {
			conn, _ := ln.Accept()
			time.Sleep(2 * time.Second)
			if conn != nil {
				_ = conn.Close()
			}
		}()

		clientConn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		sock := NewFramedTCPSocket(clientConn, 500*time.Millisecond)

		buf := make([]byte, 1024)
		_, err = sock.Read(buf)

		if err == nil {
			t.Error("Expected error on silent zombie, got nil")
		} else if !strings.Contains(err.Error(), "timeout") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})

	t.Run("Infinite_Zombie_Wait", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()

		go func() {
			conn, _ := ln.Accept()
			time.Sleep(2 * time.Second)
			if conn != nil {
				_ = conn.Close()
			}
		}()

		clientConn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		sock := NewFramedTCPSocket(clientConn, 0)

		done := make(chan bool, 1)
		go func() {
			buf := make([]byte, 1024)
			_, _ = sock.Read(buf)
			done <- true
		}()

		select {
		case <-done:
			t.Fatal("Connection closed too early! Should have waited forever.")
		case <-time.After(1 * time.Second):
			// Success: Connection is still waiting
			_ = sock.Close()
		}
	})

	t.Run("SHM_Forever_Wait", func(t *testing.T) {
		tempName := "shm_forever_test_final.tmp"
		_ = os.Remove(tempName)

		f, err := os.Create(tempName)
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Truncate(int64(TotalSize))
		f.Close()

		f2, err := os.OpenFile(tempName, os.O_RDWR, 0666)
		if err != nil {
			t.Fatal(err)
		}

		m, err := mmap.Map(f2, mmap.RDWR, 0)
		if err != nil {
			f2.Close()
			t.Fatal(err)
		}

		shmServer := NewShmTransport(f2, m, 100*time.Millisecond)
		_ = shmServer.SetIdleTimeout(0)

		done := make(chan bool, 1)
		go func() {
			buf := make([]byte, 10)
			_, _ = shmServer.Read(buf)
			done <- true
		}()

		select {
		case <-done:
			t.Fatal("SHM closed too early! Should have waited forever.")
		case <-time.After(500 * time.Millisecond):
			// Success: Still spinning
			_ = shmServer.Close()
			// Wait for the reader goroutine to actually exit before we delete the file
			<-done
		}

		f2.Close()
		time.Sleep(50 * time.Millisecond) // Final OS cleanup breather
		_ = os.Remove(tempName)
	})
}
