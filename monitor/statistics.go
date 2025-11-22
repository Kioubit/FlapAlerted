package monitor

import (
	"FlapAlerted/bgp/common"
	"math"
	"slices"
	"sort"
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
		sum = SafeAddUint64(sum, u)
	}
	avg := float64(sum) / float64(len(changesList))
	return avg / 5 // Data collected for 5 seconds
}

func GetSessionInfoJson() (string, error) {
	return common.GetSessionInfoJson()
}

//-------------------------------------------------------

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
		time.Sleep(5 * time.Second)

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
			Changes:       GlobalTotalRouteChangeCounter.Swap(0),
			ListedChanges: GlobalListedRouteChangeCounter.Swap(0),
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
