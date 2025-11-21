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
	RouterID             netip.Addr
	HasExtendedNextHopV4 bool
}

// Established session tracker
var (
	SessionTracker     = make(map[net.Conn]establishedSession)
	SessionTrackerLock sync.RWMutex
)

type establishedSession struct {
	Remote   string
	RouterID string
	Hostname string
	Time     int64
	session  *LocalSession
}

func AddSession(conn net.Conn, routerId string, hostname string, session *LocalSession) {
	newSession := establishedSession{
		Remote:   conn.RemoteAddr().String(),
		RouterID: routerId,
		Hostname: hostname,
		Time:     time.Now().Unix(),
		session:  session,
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
	var sessions = make([]establishedSession, 0)
	for _, session := range SessionTracker {
		sessions = append(sessions, session)
	}
	result, err := json.Marshal(sessions)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
