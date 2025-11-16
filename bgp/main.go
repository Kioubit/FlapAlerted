package bgp

import (
	"FlapAlerted/bgp/update"
	"FlapAlerted/config"
	"log/slog"
	"net"
	"os"
	"sync"
)

/*
- RFCs implemented:
- "A Border Gateway Protocol 4 (BGP-4)" https://datatracker.ietf.org/doc/html/rfc4271
- "Multiprotocol Extensions for BGP-4" https://datatracker.ietf.org/doc/html/rfc4760
- "Advertisement of Multiple Paths in BGP" https://datatracker.ietf.org/doc/html/rfc7911
- "BGP Support for Four-Octet Autonomous System (AS) Number Space" https://datatracker.ietf.org/doc/html/rfc6793
- "Capabilities Advertisement with BGP-4" https://datatracker.ietf.org/doc/html/rfc3392
- "Extended Message Support for BGP" https://datatracker.ietf.org/doc/html/rfc8654
- "Hostname Capability for BGP" https://datatracker.ietf.org/doc/html/draft-walton-bgp-hostname-capability-02
*/

func StartBGP(updateChannel chan update.Msg, bgpListenAddress string) {
	listener, err := net.Listen("tcp", bgpListenAddress)
	if err != nil {
		slog.Error("Failed to start BGP listener", "error", err)
		os.Exit(1)
	}
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)
	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("Failed to accept TCP connection", "error", err.Error())
			continue
		}

		logger := slog.With("remote", conn.RemoteAddr())
		logger.Info("New connection")

		go func() {
			err, wasEstablished := newBGPConnection(logger, conn, update.AFI4, config.GlobalConf.UseAddPath, config.GlobalConf.Asn,
				config.GlobalConf.RouterID, updateChannel)
			if err != nil {
				logger.Error("connection encountered an error", "error", err.Error())
			}
			_ = conn.Close()
			if wasEstablished {
				sessionCount(false)
			}
		}()
	}
}

// Established session counter
var (
	SessionCounter     int
	SessionCounterLock sync.RWMutex
)

func sessionCount(add bool) {
	SessionCounterLock.Lock()
	defer SessionCounterLock.Unlock()
	if add {
		SessionCounter++
	} else {
		if SessionCounter > 0 {
			SessionCounter--
		}
	}
}

func GetSessionCount() int {
	SessionCounterLock.RLock()
	defer SessionCounterLock.RUnlock()
	return SessionCounter
}
