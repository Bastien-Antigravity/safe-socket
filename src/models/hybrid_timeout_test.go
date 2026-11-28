package models

import (
	"testing"
)

func TestResolveHybridTimeouts_SHMAndUDP(t *testing.T) {
	d, hb := ResolveHybridTimeouts("shm-hello", "server", false)
	if d != 0 || hb != 0 {
		t.Fatalf("SHM must use infinite wait (0, 0), got (%v, %v)", d, hb)
	}
	dDocker, hbDocker := ResolveHybridTimeouts("shm", "client", true)
	if dDocker != 0 || hbDocker != 0 {
		t.Fatalf("SHM in Docker must still use (0, 0), got (%v, %v)", dDocker, hbDocker)
	}

	uD, uHb := ResolveHybridTimeouts("udp-hello", "client", false)
	if uD != DefaultUDPIdleTimeout || uHb != 0 {
		t.Fatalf("UDP expected (%v, 0), got (%v, %v)", DefaultUDPIdleTimeout, uD, uHb)
	}
}

func TestResolveHybridTimeouts_LocalVsDockerTCP(t *testing.T) {
	// Local Native TCP/TLS
	srvLocalD, srvLocalHb := ResolveHybridTimeouts("tcp-hello", "server", false)
	if srvLocalD != LocalTCPServerIdleTimeout || srvLocalHb != 0 {
		t.Fatalf("Local TCP server expected (%v, 0), got (%v, %v)", LocalTCPServerIdleTimeout, srvLocalD, srvLocalHb)
	}

	cliLocalD, cliLocalHb := ResolveHybridTimeouts("tls-hello", "client", false)
	if cliLocalD != LocalTCPClientIdleTimeout || cliLocalHb != LocalTCPClientHeartbeat {
		t.Fatalf("Local TLS client expected (%v, %v), got (%v, %v)", LocalTCPClientIdleTimeout, LocalTCPClientHeartbeat, cliLocalD, cliLocalHb)
	}

	// Docker Bridge/Overlay TCP/TLS
	srvDocD, srvDocHb := ResolveHybridTimeouts("tcp-hello", "server", true)
	if srvDocD != DockerTCPServerIdleTimeout || srvDocHb != 0 {
		t.Fatalf("Docker TCP server expected (%v, 0), got (%v, %v)", DockerTCPServerIdleTimeout, srvDocD, srvDocHb)
	}

	cliDocD, cliDocHb := ResolveHybridTimeouts("tcp-hello", "client", true)
	if cliDocD != DockerTCPClientIdleTimeout || cliDocHb != DockerTCPClientHeartbeat {
		t.Fatalf("Docker TCP client expected (%v, %v), got (%v, %v)", DockerTCPClientIdleTimeout, DockerTCPClientHeartbeat, cliDocD, cliDocHb)
	}

	cfg := NewHybridSocketConfig("tcp-hello", "client", "127.0.0.1", true)
	if cfg.DisableTimeout || cfg.Deadline != DockerTCPClientIdleTimeout || cfg.HeartbeatInterval != DockerTCPClientHeartbeat {
		t.Fatalf("Unexpected NewHybridSocketConfig: %+v", cfg)
	}

	shmCfg := NewHybridSocketConfig("shm-hello", "client", "", false)
	if !shmCfg.DisableTimeout || shmCfg.Deadline != 0 || shmCfg.HeartbeatInterval != 0 {
		t.Fatalf("Expected SHM NewHybridSocketConfig to have DisableTimeout=true and (0, 0), got %+v", shmCfg)
	}
}
