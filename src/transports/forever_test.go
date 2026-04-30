package transports

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestForeverTimeoutParity(t *testing.T) {
	// We want to test that SetIdleTimeout(0) prevents the 200ms default from firing.
	
	t.Run("TCP_Forever", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()

		go func() {
			conn, _ := ln.Accept()
			// Set a very small default, then override with 0 (forever)
			sock := NewFramedTCPSocket(conn, 100*time.Millisecond)
			_ = sock.SetIdleTimeout(0)
			
			// Wait much longer than the default 100ms
			time.Sleep(500 * time.Millisecond)
			
			buf := make([]byte, 10)
			_, _ = sock.Read(buf)
			_ = sock.Close()
		}()

		clientConn, _ := net.Dial("tcp", ln.Addr().String())
		clientSock := NewFramedTCPSocket(clientConn, 0)
		
		// Wait 400ms (4x the original default)
		time.Sleep(400 * time.Millisecond)
		
		_, err = clientSock.Write([]byte("hello"))
		assert.NoError(t, err, "TCP should not timeout after SetIdleTimeout(0)")
		_ = clientSock.Close()
	})

	t.Run("UDP_Forever", func(t *testing.T) {
		addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		serverConn, _ := net.ListenUDP("udp", addr)
		defer serverConn.Close()

		go func() {
			// Set a very small default, then override with 0 (forever)
			sock := NewUdpSocket(serverConn, 100*time.Millisecond)
			_ = sock.SetIdleTimeout(0)
			
			// Wait much longer than the default 100ms
			time.Sleep(500 * time.Millisecond)
			
			buf := make([]byte, 1024)
			_, _ = sock.Read(buf)
		}()

		clientConn, _ := net.DialUDP("udp", nil, serverConn.LocalAddr().(*net.UDPAddr))
		clientSock := NewUdpSocket(clientConn, 0)
		
		// Wait 400ms
		time.Sleep(400 * time.Millisecond)
		
		_, err := clientSock.Write([]byte("hello"))
		assert.NoError(t, err, "UDP should not timeout after SetIdleTimeout(0)")
		_ = clientSock.Close()
	})
}
