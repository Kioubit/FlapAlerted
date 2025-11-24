package monitor

import (
	"FlapAlerted/bgp/table"
	"FlapAlerted/config"
	"errors"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

var hasUserDefined atomic.Bool

type userDefined struct {
	event *FlapEvent
	count int
}

var (
	UserDefinedMap     = make(map[netip.Prefix]*userDefined)
	UserDefinedMapLock sync.RWMutex
)

func NewUserDefined(prefix netip.Prefix) error {
	UserDefinedMapLock.Lock()
	defer UserDefinedMapLock.Unlock()

	if val, exists := UserDefinedMap[prefix]; exists {
		val.count++
		return nil
	}

	if len(UserDefinedMap) >= config.GlobalConf.MaxUserDefined {
		return errors.New("too many user-defined tracked prefixes")
	}

	UserDefinedMap[prefix] = &userDefined{
		event: &FlapEvent{
			Prefix:           prefix,
			PathHistory:      newPathTracker(1000),
			TotalPathChanges: 0,
			RateSecHistory:   []int{},
			IsActive:         true,
			FirstSeen:        time.Now(),
		},
		count: 1,
	}
	hasUserDefined.Store(true)
	return nil
}

func RemoveUserDefined(prefix netip.Prefix) {
	UserDefinedMapLock.Lock()
	defer UserDefinedMapLock.Unlock()
	if val, exists := UserDefinedMap[prefix]; exists {
		val.count--
		if val.count == 0 {
			delete(UserDefinedMap, prefix)
		}
	}
	if len(UserDefinedMap) == 0 {
		hasUserDefined.Store(false)
	}
}

func GetUserDefinedEventCount(prefix netip.Prefix) uint64 {
	UserDefinedMapLock.RLock()
	defer UserDefinedMapLock.RUnlock()
	if val, exists := UserDefinedMap[prefix]; exists {
		return val.event.TotalPathChanges
	}
	return 0
}

func GetUserDefinedEvent(prefix netip.Prefix) *FlapEvent {
	UserDefinedMapLock.RLock()
	defer UserDefinedMapLock.RUnlock()
	if val, exists := UserDefinedMap[prefix]; exists {
		return val.event
	}
	return nil
}

func recordUserDefined(userPathChangeChan chan table.PathChange) {
	for {
		pathChange := <-userPathChangeChan
		UserDefinedMapLock.RLock()
		if val, exists := UserDefinedMap[pathChange.Prefix]; exists {
			incrementUint64(&val.event.TotalPathChanges)
			val.event.PathHistory.record(pathChange.OldPath, pathChange.IsWithdrawal)
		}
		UserDefinedMapLock.RUnlock()
	}
}
