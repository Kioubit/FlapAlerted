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
	globalTotalRouteChangeCounter  atomic.Uint64
	globalListedRouteChangeCounter atomic.Uint64
)

var sendUserDefined atomic.Bool

type FlapEvent struct {
	Prefix           netip.Prefix
	PathHistory      *PathTracker
	TotalPathChanges uint64

	RateSec           int
	lastIntervalCount uint64
	RateSecHistory    []int

	hasTriggered        bool
	FirstSeen           int64
	underThresholdCount int
	overThresholdCount  int
}

const intervalSec = 60

func recordPathChanges(pathChan, userPathChangeChan chan table.PathChange) {
	cleanupTicker := time.NewTicker(intervalSec * time.Second)
	counterMap := make(map[netip.Prefix]uint32)
	now := time.Now().Unix()

	for {
		var pathChange table.PathChange
		select {
		case t := <-cleanupTicker.C:
			now = t.Unix()
			counterMap = make(map[netip.Prefix]uint32)
			activeMapLock.Lock()
			for prefix, event := range activeMap {
				intervalCount := event.TotalPathChanges - event.lastIntervalCount
				event.RateSec = int(intervalCount / intervalSec)
				event.lastIntervalCount = event.TotalPathChanges

				event.RateSecHistory = append(event.RateSecHistory, event.RateSec)
				if len(event.RateSecHistory) > 60 {
					event.RateSecHistory = event.RateSecHistory[1:]
				}

				if intervalCount < uint64(config.GlobalConf.RouteChangeCounter) {
					if event.hasTriggered {
						if event.underThresholdCount == config.GlobalConf.UnderThresholdTarget {
							delete(activeMap, prefix)
							notificationEndChannel <- *event
						} else {
							event.underThresholdCount++
						}
					} else {
						delete(activeMap, prefix)
					}
				} else {
					event.underThresholdCount = 0
					if event.overThresholdCount == config.GlobalConf.OverThresholdTarget {
						event.hasTriggered = true
						event.overThresholdCount++
						notificationStartChannel <- *event
					} else {
						event.overThresholdCount++
					}
				}
			}
			activeMapLock.Unlock()
			continue
		case pathChange = <-pathChan:
		}

		if sendUserDefined.Load() {
			userPathChangeChan <- pathChange
		}

		globalTotalRouteChangeCounter.Add(1)

		activeMapLock.Lock()
		if val, exists := activeMap[pathChange.Prefix]; exists {
			incrementUint64(&val.TotalPathChanges)
			val.PathHistory.record(pathChange.OldPath, pathChange.IsWithdrawal)
			if val.hasTriggered {
				globalListedRouteChangeCounter.Add(1)
			}
		} else {
			if counterMap[pathChange.Prefix] == uint32(config.GlobalConf.RouteChangeCounter) {
				activeMap[pathChange.Prefix] = &FlapEvent{
					Prefix:             pathChange.Prefix,
					PathHistory:        newPathTracker(pathHistoryLimit),
					TotalPathChanges:   uint64(counterMap[pathChange.Prefix]) + 1,
					RateSec:            -1,
					RateSecHistory:     make([]int, 0),
					FirstSeen:          now,
					overThresholdCount: 1,
					// Special case for the 'display all route changes' mode
					hasTriggered: config.GlobalConf.RouteChangeCounter == 0,
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

func safeAddUint64(a, b uint64) uint64 {
	sum := a + b
	if sum >= a {
		return sum
	}
	return math.MaxUint64
}
