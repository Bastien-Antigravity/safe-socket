package utils

import (
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// MachineDetector determines whether an address (IP or hostname) resolves to the local machine
// or requires cross-machine communication.
type MachineDetector struct {
	mu           sync.RWMutex
	localIPs     map[string]bool
	hostname     string
	lastResolved time.Time
	ttl          time.Duration
}

var (
	defaultDetector *MachineDetector
	detectorOnce    sync.Once
)

// GetMachineDetector returns the singleton MachineDetector instance.
func GetMachineDetector() *MachineDetector {
	detectorOnce.Do(func() {
		defaultDetector = &MachineDetector{
			localIPs: make(map[string]bool),
			ttl:      30 * time.Second, // Refresh dynamic NIC interfaces periodically
		}
		defaultDetector.refresh()
	})
	return defaultDetector
}

func (d *MachineDetector) refresh() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.localIPs = make(map[string]bool)

	// Standard local addresses
	d.localIPs["127.0.0.1"] = true
	d.localIPs["127.0.0.2"] = true
	d.localIPs["::1"] = true
	d.localIPs["0.0.0.0"] = true
	d.localIPs["localhost"] = true

	// Hostname resolution
	if host, err := os.Hostname(); err == nil {
		d.hostname = strings.ToLower(host)
		d.localIPs[d.hostname] = true
	}

	// Enumerate all active network interface addresses on this machine
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				d.localIPs[ipNet.IP.String()] = true
			}
		}
	}

	d.lastResolved = time.Now()
}

// IsLocalAddress checks if an address ("IP:Port", "host:Port", or "IP") belongs to the local machine.
func (d *MachineDetector) IsLocalAddress(address string) bool {
	cleanAddr := strings.TrimSpace(address)
	if cleanAddr == "" {
		return true
	}

	// Strip port if present
	host, _, err := net.SplitHostPort(cleanAddr)
	if err != nil {
		host = cleanAddr
	}
	host = strings.TrimSpace(strings.ToLower(host))

	// Fast-path for loopbacks
	if strings.HasPrefix(host, "127.") || host == "::1" || host == "localhost" || host == "0.0.0.0" {
		return true
	}

	// Check cached local IPs
	d.mu.RLock()
	if time.Since(d.lastResolved) > d.ttl {
		d.mu.RUnlock()
		d.refresh()
		d.mu.RLock()
	}
	isLocal := d.localIPs[host]
	d.mu.RUnlock()

	if isLocal {
		return true
	}

	// If hostname is provided (e.g. Docker alias, mDNS), perform DNS lookup
	if ips, err := net.LookupIP(host); err == nil {
		d.mu.RLock()
		defer d.mu.RUnlock()
		for _, ip := range ips {
			if d.localIPs[ip.String()] {
				return true
			}
		}
	}

	return false
}
