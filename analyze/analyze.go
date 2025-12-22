package analyze

import (
	"FlapAlerted/bgp/table"
	"FlapAlerted/config"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

var (
	activeMap     = make(map[netip.Prefix]*FlapEvent)
	activeMapLock sync.RWMutex
)

var (
	GlobalTotalRouteChangeCounter  atomic.Uint64
	GlobalListedRouteChangeCounter atomic.Uint64
)

var sendUserDefined atomic.Bool

type FlapEvent struct {
	// ===== Core data =====
	Prefix           netip.Prefix
	PathHistory      *PathTracker
	TotalPathChanges uint64

	// ===== Rate calculation =====
	RateSecHistory    []int
	lastIntervalCount uint64
	RateSec           int

	// ===== State tracking =====
	FirstSeen           int64
	overThresholdCount  int
	underThresholdCount int
	hasTriggered        bool
}

type FlapEventNotification struct {
	Event   FlapEvent
	IsStart bool
}

const intervalSec = 60

func RecordPathChanges(pathChan <-chan table.PathChange) (<-chan table.PathChange, <-chan []FlapEventNotification) {
	userPathChangeChan := make(chan table.PathChange, 1000)
	notificationChannel := make(chan []FlapEventNotification, 5)

	notificationsBatch := make([]FlapEventNotification, 0, 5)

	go func() {
		defer close(userPathChangeChan)
		defer close(notificationChannel)

		cleanupTicker := time.NewTicker(intervalSec * time.Second)
		counterMap := make(map[netip.Prefix]uint32)
		now := time.Now().Unix()

		lc := 0

		for {
			var pathChange table.PathChange
			var ok bool
			select {
			case t := <-cleanupTicker.C:
				now = t.Unix()
				if lc > 30 {
					lc = 0
					counterMap = make(map[netip.Prefix]uint32)
				} else {
					lc++
					clear(counterMap)
				}
				activeMapLock.Lock()
				for prefix, event := range activeMap {
					intervalCount := event.TotalPathChanges - event.lastIntervalCount
					event.RateSec = int(intervalCount / intervalSec)
					event.lastIntervalCount = event.TotalPathChanges

					event.RateSecHistory = append(event.RateSecHistory, event.RateSec)
					if len(event.RateSecHistory) > 60 {
						event.RateSecHistory = event.RateSecHistory[1:]
					}

					if intervalCount <= uint64(config.GlobalConf.RouteChangeCounter) {
						if event.hasTriggered {
							if intervalCount <= uint64(config.GlobalConf.ExpiryRouteChangeCounter) {
								if event.underThresholdCount == config.GlobalConf.UnderThresholdTarget {
									delete(activeMap, prefix)
									notificationsBatch = append(notificationsBatch, FlapEventNotification{
										IsStart: false,
										Event:   copyEvent(event),
									})
								} else {
									event.underThresholdCount++
								}
							}
						} else {
							delete(activeMap, prefix)
						}
					} else {
						event.underThresholdCount = 0
						if event.overThresholdCount == config.GlobalConf.OverThresholdTarget {
							event.hasTriggered = true
							event.overThresholdCount++
							notificationsBatch = append(notificationsBatch, FlapEventNotification{
								IsStart: true,
								Event:   copyEvent(event),
							})
						} else {
							event.overThresholdCount++
						}
					}
				}
				activeMapLock.Unlock()
				if len(notificationsBatch) > 0 {
					select {
					case notificationChannel <- notificationsBatch:
					default:
					}
					notificationsBatch = make([]FlapEventNotification, 0, 5)
				}
				continue
			case pathChange, ok = <-pathChan:
				if !ok {
					return
				}
			}

			if sendUserDefined.Load() {
				select {
				case userPathChangeChan <- pathChange:
				default:
				}
			}

			GlobalTotalRouteChangeCounter.Add(1)

			activeMapLock.Lock()
			if val, exists := activeMap[pathChange.Prefix]; exists {
				incrementUint64(&val.TotalPathChanges)
				val.PathHistory.record(pathChange.OldPath, pathChange.IsWithdrawal)
				if val.hasTriggered {
					GlobalListedRouteChangeCounter.Add(1)
				}
			} else {
				if counterMap[pathChange.Prefix] == uint32(config.GlobalConf.RouteChangeCounter) {
					activeMap[pathChange.Prefix] = &FlapEvent{
						Prefix:             pathChange.Prefix,
						PathHistory:        newPathTracker(config.GlobalConf.MaxPathHistory),
						TotalPathChanges:   uint64(counterMap[pathChange.Prefix]) + 1,
						RateSec:            -1,
						RateSecHistory:     make([]int, 0, 1),
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
	}()
	return userPathChangeChan, notificationChannel
}
