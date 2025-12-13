package common

import (
	"encoding/json"
	"net"
	"net/netip"
	"sync"
	"time"
)

type LocalSession struct {
	DefaultAFI           AFI
	AddPathEnabled       bool
	Asn                  uint32
	OwnRouterID          netip.Addr
	RemoteRouterID       netip.Addr
	RemoteHostname       string
	HasExtendedNextHopV4 bool
	HasExtendedMessages  bool
	ApplicableHoldTime   int
}

// Established session tracker
var (
	SessionTracker     = make(map[net.Conn]establishedSession)
	SessionTrackerLock sync.RWMutex
)

type establishedSession struct {
	Remote        string
	EstablishTime int64
	session       *LocalSession
}

func AddSession(conn net.Conn, session *LocalSession) {
	newSession := establishedSession{
		Remote:        conn.RemoteAddr().String(),
		EstablishTime: time.Now().Unix(),
		session:       session,
	}
	SessionTrackerLock.Lock()
	defer SessionTrackerLock.Unlock()
	SessionTracker[conn] = newSession
}

func RemoveSession(conn net.Conn) {
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
	type JSONInfo struct {
		Remote        string
		RouterID      string
		Hostname      string
		EstablishTime int64
	}

	var sessions = make([]JSONInfo, 0)
	for _, session := range SessionTracker {
		sessions = append(sessions, JSONInfo{
			Remote:        session.Remote,
			RouterID:      session.session.RemoteRouterID.String(),
			Hostname:      session.session.RemoteHostname,
			EstablishTime: session.EstablishTime,
		})
	}
	result, err := json.Marshal(sessions)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
