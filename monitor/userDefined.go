package monitor

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/table"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

var (
	userDefinedMap     = make(map[netip.Prefix]*FlapEvent)
	userDefinedMapLock sync.RWMutex
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

func NewUserDefinedMonitor(prefix netip.Prefix) (chan UserDefinedMonitorStatistic, error) {
	userDefinedClientsLock.Lock()
	defer userDefinedClientsLock.Unlock()
	userDefinedMapLock.Lock()
	defer userDefinedMapLock.Unlock()

	if _, exists := userDefinedMap[prefix]; !exists {
		userDefinedMap[prefix] = &FlapEvent{
			Prefix:           prefix,
			PathHistory:      newPathTracker(1000),
			TotalPathChanges: 0,
			RateSecHistory:   []int{},
			hasTriggered:     true,
			FirstSeen:        time.Now().Unix(),
		}
	}

	newChannel := make(chan UserDefinedMonitorStatistic, 10)

	if val, exists := userDefinedClientsMap[prefix]; exists {
		userDefinedClientsMap[prefix] = append(val, newChannel)
	} else {
		userDefinedClientsMap[prefix] = []chan UserDefinedMonitorStatistic{newChannel}
	}
	userDefinedClientWorker()

	sendUserDefined.Store(true)
	return newChannel, nil
}

func RemoveUserDefinedMonitor(prefix netip.Prefix, channel chan UserDefinedMonitorStatistic) {
	userDefinedClientsLock.Lock()
	defer userDefinedClientsLock.Unlock()
	userDefinedMapLock.Lock()
	defer userDefinedMapLock.Unlock()

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
			delete(userDefinedMap, prefix)
		}
	}

	if len(userDefinedMap) == 0 {
		sendUserDefined.Store(false)
	}
}

func GetNumberOfUserDefinedMonitorClients() int {
	userDefinedClientsLock.RLock()
	defer userDefinedClientsLock.RUnlock()
	return len(userDefinedClientsMap)
}

func getUserDefinedMonitorCount(prefix netip.Prefix) uint64 {
	if val, exists := userDefinedMap[prefix]; exists {
		return val.TotalPathChanges
	}
	return 0
}

func GetUserDefinedMonitorEvent(prefix netip.Prefix) *FlapEvent {
	userDefinedMapLock.RLock()
	defer userDefinedMapLock.RUnlock()
	if val, exists := userDefinedMap[prefix]; exists {
		return val
	}
	return nil
}

func recordUserDefinedMonitors(userPathChangeChan <-chan table.PathChange) {
	for {
		pathChange, ok := <-userPathChangeChan
		if !ok {
			return
		}
		userDefinedMapLock.Lock()
		if val, exists := userDefinedMap[pathChange.Prefix]; exists {
			incrementUint64(&val.TotalPathChanges)
			val.PathHistory.record(pathChange.OldPath, pathChange.IsWithdrawal)
		}
		userDefinedMapLock.Unlock()
	}
}

func userDefinedClientWorker() {
	if !userDefinedClientWorkerRunning.CompareAndSwap(false, true) {
		return
	}
	go func() {
		for {
			shouldExit := func() bool {
				userDefinedClientsLock.RLock()
				userDefinedMapLock.RLock()
				defer userDefinedMapLock.RUnlock()
				defer userDefinedClientsLock.RUnlock()

				if len(userDefinedClientsMap) == 0 {
					userDefinedClientWorkerRunning.Store(false)
					return true
				}
				bgpSessionCnt := common.GetSessionCount()
				for prefix, clients := range userDefinedClientsMap {
					uds := UserDefinedMonitorStatistic{
						Count:    getUserDefinedMonitorCount(prefix),
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
