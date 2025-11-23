package monitor

import (
	"FlapAlerted/bgp/common"
	"math"
	"slices"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func GetAverageRouteChanges90() float64 {
	stats := GetStats()
	changesList := make([]uint64, len(stats))
	for i, stat := range stats {
		changesList[i] = stat.Stats.Changes
	}
	sort.Slice(changesList, func(i, j int) bool { return changesList[i] < changesList[j] })
	cutLength := int(math.Ceil(float64(len(changesList)) * 0.90))
	changesList = changesList[:cutLength]

	if len(changesList) == 0 {
		return 0
	}

	var sum uint64 = 0
	for _, u := range changesList {
		sum = safeAddUint64(sum, u)
	}
	avg := float64(sum) / float64(len(changesList))
	return avg / statisticsCollectionIntervalSec
}

func GetSessionInfoJson() (string, error) {
	return common.GetSessionInfoJson()
}

func GetActiveFlaps() []FlapEvent {
	active, _ := GetActiveFlapList()
	return active
}

func GetActiveFlapsSummary() []FlapSummary {
	l := lastFlapSummaryList.Load()
	if l == nil {
		return make([]FlapSummary, 0)
	}
	return *l
}

type Metric struct {
	ActiveFlapCount                int
	ActiveFlapTotalPathChangeCount uint64
	AverageRouteChanges90          string
	Sessions                       int
}

func GetMetric() Metric {
	var activeFlapCount = 0
	var pathChangeCount uint64 = 0
	stats := GetStats()
	if len(stats) != 0 {
		activeFlapCount = stats[len(stats)-1].Stats.Active
		pathChangeCount = stats[len(stats)-1].Stats.Changes
	}
	avg := GetAverageRouteChanges90()
	avgStr := strconv.FormatFloat(avg, 'f', 2, 64)

	return Metric{
		ActiveFlapCount:                activeFlapCount,
		ActiveFlapTotalPathChangeCount: pathChangeCount,
		AverageRouteChanges90:          avgStr,
		Sessions:                       common.GetSessionCount(),
	}
}

// -------------------------------------------------------
const statisticsCollectionIntervalSec = 5

type statisticWrapper struct {
	List     []FlapSummary
	Stats    statistic
	Sessions int
}
type FlapSummary struct {
	Prefix     string
	FirstSeen  int64
	RateSec    int
	TotalCount uint64
}

type statistic struct {
	Time          int64
	Changes       uint64
	ListedChanges uint64
	Active        int
}

var (
	statList     = make([]statistic, 0)
	statListLock sync.RWMutex

	statSubscribers     = make([]chan statisticWrapper, 0)
	statSubscribersLock sync.Mutex
)

func addStatSubscriber() chan statisticWrapper {
	statSubscribersLock.Lock()
	defer statSubscribersLock.Unlock()
	c := make(chan statisticWrapper, 2)
	statSubscribers = append(statSubscribers, c)
	return c
}

var lastFlapSummaryList atomic.Pointer[[]FlapSummary]

func statTracker() {
	for {
		time.Sleep(statisticsCollectionIntervalSec * time.Second)

		aFlap, trackedCount := GetActiveFlapList()

		if len(aFlap) > 100 {
			slices.SortFunc(aFlap, func(a, b FlapEvent) int {
				if b.TotalPathChanges > a.TotalPathChanges {
					return 1
				} else if b.TotalPathChanges < a.TotalPathChanges {
					return -1
				}
				return 0
			})
			aFlap = aFlap[:100]
		}

		jsFlapList := make([]FlapSummary, len(aFlap))
		for i, f := range aFlap {
			jsFlapList[i] = FlapSummary{
				Prefix:     f.Prefix.String(),
				FirstSeen:  f.FirstSeen.Unix(),
				RateSec:    f.RateSec,
				TotalCount: f.TotalPathChanges,
			}
		}

		lastFlapSummaryList.Store(&jsFlapList)

		newStatistic := statistic{
			Time:          time.Now().Unix(),
			Changes:       globalTotalRouteChangeCounter.Swap(0),
			ListedChanges: globalListedRouteChangeCounter.Swap(0),
			Active:        trackedCount,
		}

		newWrapper := statisticWrapper{
			List:     jsFlapList,
			Stats:    newStatistic,
			Sessions: common.GetSessionCount(),
		}

		statSubscribersLock.Lock()
		for _, subscriber := range statSubscribers {
			select {
			case subscriber <- newWrapper:
			default:
			}
		}
		statSubscribersLock.Unlock()

		statListLock.Lock()
		statList = append(statList, newStatistic)
		if len(statList) > 50 {
			statList = statList[1:]
		}
		statListLock.Unlock()
	}
}

func GetStats() []statisticWrapper {
	statListLock.RLock()
	defer statListLock.RUnlock()
	result := make([]statisticWrapper, len(statList))
	for i := range statList {
		result[i] = statisticWrapper{
			List:     nil,
			Stats:    statList[i],
			Sessions: -1,
		}
	}
	if len(statList) > 0 {
		l := lastFlapSummaryList.Load()
		if l != nil {
			result[len(statList)-1].List = *l
		}
	}
	return result
}

func SubscribeToStats() chan statisticWrapper {
	return addStatSubscriber()
}
