package transports

// =============================================================================
// ESSENTIAL PROCESS:
// Unit tests verifying framed TCP connection read/write symmetry, payload integrity
// across varying buffer sizes, consecutive frame streams, and short buffer recovery.
//
// DATA FLOW:
// 1. Input: Byte payloads spanning edge cases (0B, 1B, 1KB, 64KB, 1MB).
// 2. Logic: Writes frames through FramedTCPSocket and verifies Read & ReadMessage output.
// 3. Output: Pass/fail assertions validating wire format parity and zero regression.
//
// KEY PARAMETERS:
// - t: Testing handle.
// =============================================================================

import (
	"bytes"
	"crypto/rand"
	"io"
	"net"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------

func TestFramedTCP_WriteRead_VaryingSizes(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	testPayloadSizes := []int{1, 4, 128, 1024, 65536, 524288}

	for _, size := range testPayloadSizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			payload := make([]byte, size)
			_, _ = rand.Read(payload)

			done := make(chan struct{})
			go func() {
				defer close(done)
				conn, err := ln.Accept()
				if err != nil {
					t.Errorf("Server accept error: %v", err)
					return
				}
				defer func() { _ = conn.Close() }()

				serverSock := NewFramedTCPSocket(conn, 2*time.Second)

				// 1. Read with Read()
				buf := make([]byte, size+100)
				n, err := serverSock.Read(buf)
				if err != nil {
					t.Errorf("Server Read failed: %v", err)
					return
				}
				if n != size {
					t.Errorf("Expected %d bytes read, got %d", size, n)
					return
				}
				if !bytes.Equal(buf[:n], payload) {
					t.Errorf("Payload mismatch on Read")
					return
				}

				// 2. Echo back with ReadMessage counterpart
				msg, err := serverSock.ReadMessage()
				if err != nil {
					t.Errorf("Server ReadMessage failed: %v", err)
					return
				}
				if len(msg) != size {
					t.Errorf("Expected ReadMessage size %d, got %d", size, len(msg))
					return
				}
				if !bytes.Equal(msg, payload) {
					t.Errorf("Payload mismatch on ReadMessage")
					return
				}

				// Echo back
				wN, err := serverSock.Write(msg)
				if err != nil {
					t.Errorf("Server Write failed: %v", err)
					return
				}
				if wN != size {
					t.Errorf("Server Write returned %d, expected %d", wN, size)
					return
				}
			}()

			clientConn, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatalf("Client dial failed: %v", err)
			}
			defer func() { _ = clientConn.Close() }()

			clientSock := NewFramedTCPSocket(clientConn, 2*time.Second)

			// 1. Client writes payload for server.Read()
			n, err := clientSock.Write(payload)
			if err != nil {
				t.Fatalf("Client Write 1 failed: %v", err)
			}
			if n != size {
				t.Fatalf("Client Write returned %d, expected %d", n, size)
			}

			// 2. Client writes payload for server.ReadMessage()
			n, err = clientSock.Write(payload)
			if err != nil {
				t.Fatalf("Client Write 2 failed: %v", err)
			}
			if n != size {
				t.Fatalf("Client Write returned %d, expected %d", n, size)
			}

			// 3. Client reads echo
			echoBuf := make([]byte, size)
			echoN, err := clientSock.Read(echoBuf)
			if err != nil {
				t.Fatalf("Client Read echo failed: %v", err)
			}
			if echoN != size {
				t.Fatalf("Echo byte count %d != size %d", echoN, size)
			}
			if !bytes.Equal(echoBuf, payload) {
				t.Fatalf("Echo payload mismatch")
			}

			<-done
		})
	}
}

// -----------------------------------------------------------------------------

func TestFramedTCP_ConsecutiveFrames(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	const frameCount = 100
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			t.Errorf("Server accept error: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

		serverSock := NewFramedTCPSocket(conn, 2*time.Second)
		for i := 0; i < frameCount; i++ {
			msg, err := serverSock.ReadMessage()
			if err != nil {
				t.Errorf("Frame %d ReadMessage error: %v", i, err)
				return
			}
			expected := []byte{byte(i % 256), 0xAA, 0xBB}
			if !bytes.Equal(msg, expected) {
				t.Errorf("Frame %d mismatch: got %v, want %v", i, msg, expected)
				return
			}
		}
	}()

	clientConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Client dial failed: %v", err)
	}
	defer func() { _ = clientConn.Close() }()

	clientSock := NewFramedTCPSocket(clientConn, 2*time.Second)
	for i := 0; i < frameCount; i++ {
		payload := []byte{byte(i % 256), 0xAA, 0xBB}
		n, err := clientSock.Write(payload)
		if err != nil {
			t.Fatalf("Write frame %d failed: %v", i, err)
		}
		if n != len(payload) {
			t.Fatalf("Write frame %d returned n=%d, expected %d", i, n, len(payload))
		}
	}

	<-done
}

// -----------------------------------------------------------------------------

func TestFramedTCP_ShortBufferRecovery(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	payload := []byte("A_LONGER_MESSAGE_THAN_SHORT_BUFFER")

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		s := NewFramedTCPSocket(conn, 2*time.Second)
		_, _ = s.Write(payload)
	}()

	clientConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Client dial failed: %v", err)
	}
	defer func() { _ = clientConn.Close() }()

	clientSock := NewFramedTCPSocket(clientConn, 2*time.Second)

	// 1. Attempt to read into too small buffer
	tooShort := make([]byte, 5)
	_, err = clientSock.Read(tooShort)
	if err != io.ErrShortBuffer {
		t.Fatalf("Expected io.ErrShortBuffer, got %v", err)
	}

	// 2. Retry with adequate buffer - header must NOT have been lost!
	properBuf := make([]byte, 100)
	n, err := clientSock.Read(properBuf)
	if err != nil {
		t.Fatalf("Retry read failed: %v", err)
	}
	if !bytes.Equal(properBuf[:n], payload) {
		t.Fatalf("Payload mismatch after short buffer recovery: got %q, want %q", properBuf[:n], payload)
	}
}
