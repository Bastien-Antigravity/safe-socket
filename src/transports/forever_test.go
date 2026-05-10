package transports

import (
	"net"
	"testing"
	"time"
)

func TestForeverTimeoutParity(t *testing.T) {
	t.Run("TCP_Forever", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()

		go func() {
			conn, _ := ln.Accept()
			sock := NewFramedTCPSocket(conn, 100*time.Millisecond)
			_ = sock.SetIdleTimeout(0)
			time.Sleep(500 * time.Millisecond)
			buf := make([]byte, 10)
			_, _ = sock.Read(buf)
			_ = sock.Close()
		}()

		clientConn, _ := net.Dial("tcp", ln.Addr().String())
		clientSock := NewFramedTCPSocket(clientConn, 0)
		time.Sleep(400 * time.Millisecond)

		_, err = clientSock.Write([]byte("hello"))
		if err != nil {
			t.Errorf("TCP should not timeout after SetIdleTimeout(0), got: %v", err)
		}
		_ = clientSock.Close()
	})

	t.Run("UDP_Forever", func(t *testing.T) {
		addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		serverConn, _ := net.ListenUDP("udp", addr)
		defer serverConn.Close()

		go func() {
			sock := NewUdpSocket(serverConn, 100*time.Millisecond)
			_ = sock.SetIdleTimeout(0)
			time.Sleep(500 * time.Millisecond)
			buf := make([]byte, 1024)
			_, _ = sock.Read(buf)
		}()

		clientConn, _ := net.DialUDP("udp", nil, serverConn.LocalAddr().(*net.UDPAddr))
		clientSock := NewUdpSocket(clientConn, 0)
		time.Sleep(400 * time.Millisecond)

		_, err := clientSock.Write([]byte("hello"))
		if err != nil {
			t.Errorf("UDP should not timeout after SetIdleTimeout(0), got: %v", err)
		}
		_ = clientSock.Close()
	})
}
