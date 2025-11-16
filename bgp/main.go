package bgp

import (
	"FlapAlerted/bgp/update"
	"FlapAlerted/config"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"
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
				removeSession(conn)
			}
		}()
	}
}

// Established session tracker
var (
	SessionTracker     = make(map[net.Conn]trackedSession)
	SessionTrackerLock sync.RWMutex
)

type trackedSession struct {
	Remote   string
	RouterID string
	Hostname string
	Time     int64
}

func addSession(conn net.Conn, routerId string, hostname string) {
	newSession := trackedSession{
		Remote:   conn.RemoteAddr().String(),
		RouterID: routerId,
		Hostname: hostname,
		Time:     time.Now().Unix(),
	}
	SessionTrackerLock.Lock()
	defer SessionTrackerLock.Unlock()
	SessionTracker[conn] = newSession
}

func removeSession(conn net.Conn) {
	SessionTrackerLock.Lock()
	defer SessionTrackerLock.Unlock()
	delete(SessionTracker, conn)
}

func GetSessionCount() int {
	SessionTrackerLock.RLock()
	defer SessionTrackerLock.RUnlock()
	return len(SessionTracker)
}

func GetSessionInfoJson() (string, error) {
	SessionTrackerLock.RLock()
	defer SessionTrackerLock.RUnlock()
	var sessions = make([]trackedSession, 0)
	for _, session := range SessionTracker {
		sessions = append(sessions, session)
	}
	result, err := json.Marshal(sessions)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
