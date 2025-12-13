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
	sessionTracker     = make(map[net.Conn]establishedSession)
	sessionTrackerLock sync.RWMutex
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
	sessionTrackerLock.Lock()
	defer sessionTrackerLock.Unlock()
	sessionTracker[conn] = newSession
}

func RemoveSession(conn net.Conn) {
	sessionTrackerLock.Lock()
	defer sessionTrackerLock.Unlock()
	delete(sessionTracker, conn)
}

func GetSessionCount() int {
	sessionTrackerLock.RLock()
	defer sessionTrackerLock.RUnlock()
	return len(sessionTracker)
}

func GetSessionInfoJson() (string, error) {
	sessionTrackerLock.RLock()
	defer sessionTrackerLock.RUnlock()
	type JSONInfo struct {
		Remote        string
		RouterID      string
		Hostname      string
		EstablishTime int64
	}

	var sessions = make([]JSONInfo, 0)
	for _, session := range sessionTracker {
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
