package analyze

import (
	"FlapAlerted/bgp/table"
	"FlapAlerted/config"
	"net/netip"
	"sync"
	"time"
)

var (
	userDefinedMap     = make(map[netip.Prefix]*FlapEvent)
	userDefinedMapLock sync.RWMutex
)

func RecordUserDefinedMonitors(userPathChangeChan <-chan table.PathChange) {
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

func AddUserDefinedPrefix(prefix netip.Prefix) {
	userDefinedMapLock.Lock()
	defer userDefinedMapLock.Unlock()

	if _, exists := userDefinedMap[prefix]; !exists {
		userDefinedMap[prefix] = &FlapEvent{
			Prefix:           prefix,
			PathHistory:      newPathTracker(config.GlobalConf.MaxPathHistory),
			TotalPathChanges: 0,
			RateSecHistory:   []int{},
			hasTriggered:     true,
			FirstSeen:        time.Now().Unix(),
		}
		sendUserDefined.Store(true)
	}
}

func RemoveUserDefinedPrefix(prefix netip.Prefix) {
	userDefinedMapLock.Lock()
	defer userDefinedMapLock.Unlock()

	delete(userDefinedMap, prefix)

	if len(userDefinedMap) == 0 {
		sendUserDefined.Store(false)
	}
}

func GetUserDefinedPathChangeCount(prefix netip.Prefix) uint64 {
	userDefinedMapLock.RLock()
	defer userDefinedMapLock.RUnlock()
	if val, exists := userDefinedMap[prefix]; exists {
		return val.TotalPathChanges
	}
	return 0
}

func GetUserDefinedMonitorEvent(prefix netip.Prefix) (event FlapEvent, found bool) {
	userDefinedMapLock.RLock()
	defer userDefinedMapLock.RUnlock()
	if val, exists := userDefinedMap[prefix]; exists {
		found = true
		event = copyEvent(val)
		return
	}
	return
}
