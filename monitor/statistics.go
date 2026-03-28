package monitor

import (
	"FlapAlerted/analyze"
	"FlapAlerted/bgp/session"
	"cmp"
	"context"
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
		sum += u
	}
	avg := float64(sum) / float64(len(changesList))
	return avg / statisticsCollectionIntervalSec
}

func GetSessionInfoJson() (string, error) {
	return session.GetSessionInfoJson()
}

type Metric struct {
	ActiveFlapCount                int
	ActiveFlapTotalPathChangeCount uint64
	TotalPathChangeCount           uint64
	AverageRouteChanges90          string
	Sessions                       int
}

func GetMetric() Metric {
	var activeFlapCount int
	var activeFlapTotalPathChangeCount uint64
	var pathChangeCount uint64
	stats := GetStats()
	if len(stats) != 0 {
		activeFlapCount = stats[len(stats)-1].Stats.Active
		activeFlapTotalPathChangeCount = stats[len(stats)-1].Stats.ListedChanges
		pathChangeCount = stats[len(stats)-1].Stats.Changes
	}
	avg := GetAverageRouteChanges90()
	avgStr := strconv.FormatFloat(avg, 'f', 2, 64)

	return Metric{
		ActiveFlapCount:                activeFlapCount,
		ActiveFlapTotalPathChangeCount: activeFlapTotalPathChangeCount,
		TotalPathChangeCount:           pathChangeCount,
		AverageRouteChanges90:          avgStr,
		Sessions:                       session.GetSessionCount(),
	}
}

// -------------------------------------------------------
const statisticsCollectionIntervalSec = 5

type statisticWrapper struct {
	List      []FlapSummary
	ListPeers []PeerSummary
	Stats     statistic
	Sessions  int
}
type FlapSummary struct {
	Prefix     string
	FirstSeen  int64
	RateSec    int
	TotalCount uint64
}

type PeerSummary struct {
	ASN        uint32
	RateSecAvg float64
	RateSec    int
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

func SubscribeToStats() chan statisticWrapper {
	statSubscribersLock.Lock()
	defer statSubscribersLock.Unlock()
	c := make(chan statisticWrapper, 2)
	statSubscribers = append(statSubscribers, c)
	return c
}

var lastFlapSummaryList atomic.Pointer[[]FlapSummary]
var lastPeerSummaryList atomic.Pointer[[]PeerSummary]

func statTracker(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(statisticsCollectionIntervalSec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}

		aFlap, trackedCount := analyze.GetActiveFlapList()
		aPeer := analyze.GetPeerRates()

		if len(aFlap) > 100 {
			slices.SortFunc(aFlap, func(a, b analyze.FlapEvent) int {
				return cmp.Compare(b.TotalPathChanges, a.TotalPathChanges)
			})
			aFlap = aFlap[:100]
		}

		if len(aPeer) > 30 {
			slices.SortFunc(aPeer, func(a, b analyze.PeerUpdateRate) int {
				return cmp.Compare(b.RateSecAvg, a.RateSecAvg)
			})
			aPeer = aPeer[:30]
		}

		jsFlapList := make([]FlapSummary, len(aFlap))
		for i, f := range aFlap {
			jsFlapList[i] = FlapSummary{
				Prefix:     f.Prefix.String(),
				FirstSeen:  f.FirstSeen,
				RateSec:    f.RateSec,
				TotalCount: f.TotalPathChanges,
			}
		}

		jsPeerList := make([]PeerSummary, len(aPeer))
		for i, p := range aPeer {
			jsPeerList[i] = PeerSummary{
				ASN:        p.PeerASN,
				RateSec:    p.RateSec,
				RateSecAvg: p.RateSecAvg,
			}
		}

		lastFlapSummaryList.Store(&jsFlapList)
		lastPeerSummaryList.Store(&jsPeerList)

		newStatistic := statistic{
			Time:          time.Now().Unix(),
			Changes:       analyze.GlobalTotalRouteChangeCounter.Swap(0),
			ListedChanges: analyze.GlobalListedRouteChangeCounter.Swap(0),
			Active:        trackedCount,
		}

		newWrapper := statisticWrapper{
			List:      jsFlapList,
			ListPeers: jsPeerList,
			Stats:     newStatistic,
			Sessions:  session.GetSessionCount(),
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
			List:      nil,
			ListPeers: nil,
			Stats:     statList[i],
			Sessions:  -1,
		}
	}
	if len(statList) > 0 {
		l := lastFlapSummaryList.Load()
		p := lastPeerSummaryList.Load()
		if l != nil {
			result[len(statList)-1].List = *l
			result[len(statList)-1].ListPeers = *p
			result[len(statList)-1].Sessions = session.GetSessionCount()
		}
	}
	return result
}

func GetActiveFlapsSummary() []FlapSummary {
	l := lastFlapSummaryList.Load()
	if l == nil {
		return make([]FlapSummary, 0)
	}
	return *l
}

func GetActivePeersSummary() []PeerSummary {
	p := lastPeerSummaryList.Load()
	if p == nil {
		return make([]PeerSummary, 0)
	}
	return *p
}
