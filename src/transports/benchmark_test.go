package transports

// =============================================================================
// ESSENTIAL PROCESS:
// Reproducible throughput and memory allocation benchmarks (-benchmem) for
// FramedTCPSocket (small zero-alloc stack frames vs. large scatter-gather frames)
// and ShmTransport ring-buffer IPC.
//
// DATA FLOW:
// 1. Input: Fixed payload slices (64B, 256B, 32KB).
// 2. Logic: Measures round-trip framing and transport read/write operations per iteration.
// 3. Output: Throughput (MB/s) and heap allocation metrics (B/op, allocs/op).
//
// KEY PARAMETERS:
// - b: Standard testing.B benchmark harness.
// =============================================================================

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------

// BenchmarkFramedTCP_SmallFrameZeroAlloc verifies zero-allocation stack framing
// on FramedTCPSocket.Write and FramedTCPSocket.Read for payloads <= 508 bytes.
func BenchmarkFramedTCP_SmallFrameZeroAlloc(b *testing.B) {
	serverConn, clientConn := net.Pipe()
	defer func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	}()

	server := NewFramedTCPSocket(serverConn, 0)
	client := NewFramedTCPSocket(clientConn, 0)

	payload := bytes.Repeat([]byte("A"), 64)
	readBuf := make([]byte, 128)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < b.N; i++ {
			if _, err := server.Read(readBuf); err != nil {
				return
			}
		}
	}()

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := client.Write(payload); err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}

	b.StopTimer()
	<-done
}

// -----------------------------------------------------------------------------

// BenchmarkFramedTCP_LargeFrameThroughput measures throughput for 32KB frames.
func BenchmarkFramedTCP_LargeFrameThroughput(b *testing.B) {
	serverConn, clientConn := net.Pipe()
	defer func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	}()

	server := NewFramedTCPSocket(serverConn, 0)
	client := NewFramedTCPSocket(clientConn, 0)

	payload := bytes.Repeat([]byte("X"), 32*1024)
	readBuf := make([]byte, 32*1024)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < b.N; i++ {
			if _, err := server.Read(readBuf); err != nil {
				return
			}
		}
	}()

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := client.Write(payload); err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}

	b.StopTimer()
	<-done
}

// -----------------------------------------------------------------------------

// BenchmarkSHM_Throughput measures point-to-point shared memory ring buffer throughput.
func BenchmarkSHM_Throughput(b *testing.B) {
	shmPath := filepath.Join(b.TempDir(), "bench_shm_ring")
	defer func() { _ = os.Remove(shmPath) }()

	ln, err := ListenShm(shmPath, 5*time.Second)
	if err != nil {
		b.Fatalf("ListenShm failed: %v", err)
	}
	defer func() { _ = ln.Close() }()

	serverConnCh := make(chan error, 1)
	payload := bytes.Repeat([]byte("S"), 256)
	readBuf := make([]byte, 512)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			serverConnCh <- err
			return
		}
		defer func() { _ = conn.Close() }()
		for i := 0; i < b.N; i++ {
			if _, err := conn.Read(readBuf); err != nil {
				serverConnCh <- err
				return
			}
		}
		serverConnCh <- nil
	}()

	client, err := ConnectShm(shmPath, 5*time.Second)
	if err != nil {
		b.Fatalf("ConnectShm failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := client.Write(payload); err != nil {
			b.Fatalf("SHM Write failed: %v", err)
		}
	}

	b.StopTimer()
	if err := <-serverConnCh; err != nil {
		b.Fatalf("SHM Server Read failed: %v", err)
	}
}
