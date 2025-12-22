package monitor

import (
	"FlapAlerted/analyze"
	"FlapAlerted/bgp/session"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

var (
	userDefinedClientsMap  = make(map[netip.Prefix][]chan UserDefinedMonitorStatistic)
	userDefinedClientsLock sync.RWMutex
)

var userDefinedClientWorkerRunning atomic.Bool

type UserDefinedMonitorStatistic struct {
	Count    uint64
	Sessions int
}

func NewUserDefinedMonitor(prefix netip.Prefix) (<-chan UserDefinedMonitorStatistic, error) {
	userDefinedClientsLock.Lock()
	defer userDefinedClientsLock.Unlock()

	newChannel := make(chan UserDefinedMonitorStatistic, 10)

	if val, exists := userDefinedClientsMap[prefix]; exists {
		userDefinedClientsMap[prefix] = append(val, newChannel)
	} else {
		analyze.AddUserDefinedPrefix(prefix)
		userDefinedClientsMap[prefix] = []chan UserDefinedMonitorStatistic{newChannel}
	}
	userDefinedClientWorker()

	return newChannel, nil
}

func RemoveUserDefinedMonitor(prefix netip.Prefix, channel <-chan UserDefinedMonitorStatistic) {
	userDefinedClientsLock.Lock()
	defer userDefinedClientsLock.Unlock()

	if val, exists := userDefinedClientsMap[prefix]; exists {
		for i, c := range val {
			if c == channel {
				updatedSlice := append(val[:i], val[i+1:]...)
				userDefinedClientsMap[prefix] = updatedSlice

				close(c)
				break
			}
		}
		if len(userDefinedClientsMap[prefix]) == 0 {
			delete(userDefinedClientsMap, prefix)
			analyze.RemoveUserDefinedPrefix(prefix)
		}
	}
}

func GetNumberOfUserDefinedMonitorClients() int {
	userDefinedClientsLock.RLock()
	defer userDefinedClientsLock.RUnlock()
	return len(userDefinedClientsMap)
}

func userDefinedClientWorker() {
	if !userDefinedClientWorkerRunning.CompareAndSwap(false, true) {
		return
	}
	go func() {
		for {
			shouldExit := func() bool {
				userDefinedClientsLock.RLock()
				defer userDefinedClientsLock.RUnlock()

				if len(userDefinedClientsMap) == 0 {
					userDefinedClientWorkerRunning.Store(false)
					return true
				}
				bgpSessionCnt := session.GetSessionCount()
				for prefix, clients := range userDefinedClientsMap {
					uds := UserDefinedMonitorStatistic{
						Count:    analyze.GetUserDefinedPathChangeCount(prefix),
						Sessions: bgpSessionCnt,
					}
					for _, client := range clients {
						select {
						case client <- uds:
						default:
						}
					}
				}
				return false
			}()
			if shouldExit {
				return
			}
			time.Sleep(5 * time.Second)
		}
	}()
}
