package utils

import (
	"net"
	"testing"
)

func TestMachineDetector(t *testing.T) {
	detector := GetMachineDetector()

	// 1. Loopback tests
	loopbacks := []string{
		"127.0.0.1",
		"127.0.0.1:8080",
		"127.0.0.2:9020",
		"127.0.0.254:3306",
		"localhost",
		"localhost:5000",
		"::1",
		"[::1]:8080",
		"0.0.0.0:8000",
	}

	for _, addr := range loopbacks {
		if !detector.IsLocalAddress(addr) {
			t.Errorf("Expected %s to be recognized as local address", addr)
		}
	}

	// 2. Remote address tests
	remotes := []string{
		"8.8.8.8",
		"8.8.8.8:53",
		"1.1.1.1:443",
		"203.0.113.195:9000",
		"google.com:443",
	}

	for _, addr := range remotes {
		if detector.IsLocalAddress(addr) {
			t.Errorf("Expected %s to be recognized as remote address", addr)
		}
	}

	// 3. Dynamic interface test (our own NIC IP should be recognized as local)
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, a := range addrs {
			if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				nicAddr := net.JoinHostPort(ipNet.IP.String(), "9999")
				if !detector.IsLocalAddress(nicAddr) {
					t.Errorf("Expected host NIC IP %s to be recognized as local", nicAddr)
				}
				break
			}
		}
	}
}
