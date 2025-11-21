package monitor

import (
	"FlapAlerted/bgp/table"
	"FlapAlerted/config"
	"math"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

const pathHistoryLimit = 1000

var (
	activeMap     = make(map[netip.Prefix]*FlapEvent)
	activeMapLock sync.RWMutex
)

var (
	GlobalTotalRouteChangeCounter  atomic.Uint64
	GlobalListedRouteChangeCounter atomic.Uint64
)

var NotificationChannel = make(chan FlapEvent)
var NotificationEndChannel = make(chan FlapEvent)

type FlapEvent struct {
	Prefix           netip.Prefix
	PathHistory      *PathTracker
	TotalPathChanges uint64 // Total path changes since first exceeding

	RateSec           int
	lastIntervalCount uint64
	RateSecHistory    []int

	IsActive            bool
	FirstSeen           time.Time
	underThresholdCount int
	overThresholdCount  int
}

const intervalSec = 60

func recordPathChanges(pathChan chan table.PathChange) {
	cleanupTicker := time.NewTicker(intervalSec * time.Second)
	counterMap := make(map[netip.Prefix]uint32)
	now := time.Now()

	for {
		var pathChange table.PathChange
		select {
		case now = <-cleanupTicker.C:
			counterMap = make(map[netip.Prefix]uint32)
			activeMapLock.Lock()
			for prefix, event := range activeMap {
				intervalCount := event.TotalPathChanges - event.lastIntervalCount
				event.RateSec = int(intervalCount / intervalSec)
				event.lastIntervalCount = event.TotalPathChanges

				event.RateSecHistory = append(event.RateSecHistory, event.RateSec)
				if len(event.RateSecHistory) > 20 {
					event.RateSecHistory = event.RateSecHistory[1:]
				}

				if intervalCount < uint64(config.GlobalConf.RouteChangeCounter) {
					if event.underThresholdCount == config.GlobalConf.UnderThresholdTarget {
						delete(activeMap, prefix)
						NotificationEndChannel <- *event
					} else {
						event.underThresholdCount++
					}
				} else {
					event.underThresholdCount = 0
					if event.overThresholdCount == config.GlobalConf.OverThresholdTarget {
						event.IsActive = true
						event.overThresholdCount++
						NotificationChannel <- *event
					} else {
						event.overThresholdCount++
					}
				}
			}
			activeMapLock.Unlock()
			continue
		case pathChange = <-pathChan:
		}

		GlobalTotalRouteChangeCounter.Add(1)

		activeMapLock.Lock()
		if val, exists := activeMap[pathChange.Prefix]; exists {
			incrementUint64(&val.TotalPathChanges)
			val.PathHistory.Record(pathChange.OldPath, pathChange.IsWithdrawal)
			if val.IsActive {
				GlobalListedRouteChangeCounter.Add(1)
			}
		} else {
			if counterMap[pathChange.Prefix] == uint32(config.GlobalConf.RouteChangeCounter) {
				activeMap[pathChange.Prefix] = &FlapEvent{
					Prefix:             pathChange.Prefix,
					PathHistory:        NewPathTracker(pathHistoryLimit),
					TotalPathChanges:   uint64(counterMap[pathChange.Prefix]) + 1,
					RateSec:            -1,
					RateSecHistory:     make([]int, 0),
					FirstSeen:          now,
					overThresholdCount: 1,
				}
			} else {
				counterMap[pathChange.Prefix]++
			}
		}
		activeMapLock.Unlock()

	}
}

func incrementUint64(n *uint64) {
	if *n == math.MaxUint64 {
		return
	}
	*n = *n + 1
}

func SafeAddUint64(a, b uint64) uint64 {
	sum := a + b
	if sum >= a {
		return sum
	}
	return math.MaxUint64
}
