package monitor

import (
	"FlapAlerted/bgp"
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/update"
	"FlapAlerted/config"
	"log"
	"log/slog"
	"math"
	"net/netip"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const PathLimit = 1000

type Flap struct {
	sync.RWMutex
	Cidr                 string
	LastPath             map[string]common.AsPathList
	Paths                map[string]*PathInfo
	pathChangeCount      uint64
	PathChangeCountTotal uint64
	FirstSeen            int64
	LastSeen             int64
	meetsMinimumAge      bool
}

type PathInfo struct {
	Path  common.AsPathList
	Count uint64
}

func StartMonitoring(conf config.UserConfig) {
	config.GlobalConf = conf
	if config.GlobalConf.RelevantAsnPosition < 0 {
		log.Fatal("Invalid RelevantAsnPosition value")
	}

	updateChannel := make(chan update.Msg, 200)
	notificationChannel := make(chan *Flap, 10)
	// Initialize activeFlapList and flapMap
	flapMap = make(map[string]*Flap)
	activeFlapList = make([]*Flap, 0)

	go notificationHandler(notificationChannel)
	go processUpdates(updateChannel, notificationChannel)
	go bgp.StartBGP(updateChannel)
	go statTracker()
	go moduleCallback()
	cleanUpFlapList()
}

var (
	flapMap          map[string]*Flap
	flapMapMu        sync.RWMutex
	activeFlapList   []*Flap
	activeFlapListMu sync.RWMutex
	currentTime      int64
)

var globalTotalRouteChangeCounter atomic.Uint64
var globalListedRouteChangeCounter atomic.Uint64

func processUpdates(updateChannel chan update.Msg, notificationChannel chan *Flap) {
	for {
		u, ok := <-updateChannel
		if !ok {
			return
		}

		slog.Debug("Received update", "update", u)

		nlRi, foundNlRi, err := u.GetMpReachNLRI()
		if err != nil {
			slog.Warn("error getting MpReachNLRI", "error", err)
			continue
		}

		asPath, err := u.GetAsPaths()
		if err != nil {
			slog.Warn("error getting ASPath", "error", err)
			continue
		}

		if len(asPath) == 0 {
			continue
		}

		flapMapMu.Lock()
		currentTime = time.Now().Unix()
		if foundNlRi {
			for i := range nlRi.NLRI {
				updateList(nlRi.NLRI[i].ToNetCidr(), asPath, notificationChannel)
			}
		}
		for i := range u.NetworkLayerReachabilityInformation {
			updateList(u.NetworkLayerReachabilityInformation[i].ToNetCidr(), asPath, notificationChannel)
		}
		flapMapMu.Unlock()
	}
}

func cleanUpFlapList() {
	for {
		time.Sleep(1 * time.Duration(config.GlobalConf.FlapPeriod) * time.Second)

		flapMapMu.Lock()
		activeFlapListMu.Lock()
		currentTime = time.Now().Unix()
		flapMap = make(map[string]*Flap)
		newActiveFlapList := make([]*Flap, 0)

		for index := range activeFlapList {
			if activeFlapList[index].LastSeen+config.GlobalConf.FlapPeriod > currentTime {
				flapMap[activeFlapList[index].Cidr] = activeFlapList[index]
				newActiveFlapList = append(newActiveFlapList, activeFlapList[index])
			}
		}

		activeFlapList = newActiveFlapList
		flapMapMu.Unlock()
		activeFlapListMu.Unlock()
	}
}

func updateList(prefix netip.Prefix, asPath []common.AsPathList, notificationChannel chan *Flap) {
	cidr := prefix.String()
	if len(asPath) == 0 {
		return
	}
	cleanPath := asPath[0] // Multiple AS paths in a single update message currently unsupported (not used by bird)

	obj := flapMap[cidr]

	if obj == nil {
		newFlap := &Flap{
			Cidr:                 cidr,
			LastSeen:             currentTime,
			FirstSeen:            currentTime,
			pathChangeCount:      0,
			PathChangeCountTotal: 0,
			LastPath:             make(map[string]common.AsPathList),
			Paths:                make(map[string]*PathInfo),
		}
		if config.GlobalConf.KeepPathInfo {
			newFlap.Paths[pathToString(cleanPath)] = &PathInfo{Path: cleanPath, Count: 1}
		}
		newFlap.LastPath[getRelevantASN(cleanPath)] = cleanPath
		flapMap[cidr] = newFlap

		// Handle every Update
		if config.GlobalConf.RouteChangeCounter == 0 {
			// Only increment the global route change counter in cases where we want to show each update instead of
			// only route changes
			globalListedRouteChangeCounter.Add(1)
			newFlap.PathChangeCountTotal = 1
			newFlap.pathChangeCount = 1
			activeFlapListMu.Lock()
			activeFlapList = append(activeFlapList, newFlap)
			activeFlapListMu.Unlock()
			select {
			case notificationChannel <- newFlap:
			default:
			}

		}
		return
	}
	obj.Lock()
	defer obj.Unlock()

	// If the entry already exists

	if !pathsEqual(obj.LastPath[getRelevantASN(cleanPath)], cleanPath) {
		if config.GlobalConf.KeepPathInfo {
			if len(obj.Paths) <= PathLimit {
				searchPath := obj.Paths[pathToString(cleanPath)]
				if searchPath == nil {
					obj.Paths[pathToString(cleanPath)] = &PathInfo{Path: cleanPath, Count: 1}
				} else {
					s := obj.Paths[pathToString(cleanPath)]
					s.Count = incrementUint64(s.Count)
				}
			}
		}

		if len(obj.LastPath[getRelevantASN(cleanPath)].Asn) == 0 {
			obj.LastPath[getRelevantASN(cleanPath)] = cleanPath
			return
		}

		obj.pathChangeCount = incrementUint64(obj.pathChangeCount)
		obj.PathChangeCountTotal = incrementUint64(obj.PathChangeCountTotal)
		globalTotalRouteChangeCounter.Add(1)

		obj.LastSeen = currentTime
		obj.LastPath[getRelevantASN(cleanPath)] = cleanPath

		if obj.PathChangeCountTotal >= uint64(config.GlobalConf.RouteChangeCounter) {
			if obj.LastSeen-obj.FirstSeen > int64(config.GlobalConf.MinimumAge) {
				obj.meetsMinimumAge = true
				globalListedRouteChangeCounter.Add(1)
			}
		}

		if config.GlobalConf.RouteChangeCounter != 0 {
			if obj.pathChangeCount == uint64(config.GlobalConf.RouteChangeCounter) {
				if obj.PathChangeCountTotal == uint64(config.GlobalConf.RouteChangeCounter) {
					activeFlapListMu.Lock()
					activeFlapList = append(activeFlapList, obj)
					activeFlapListMu.Unlock()
				}
				obj.pathChangeCount = 0
				select {
				case notificationChannel <- obj:
				default:
				}
			}
		}
	}
}

func getRelevantASN(asPath common.AsPathList) string {
	pathLen := len(asPath.Asn)

	if pathLen < config.GlobalConf.RelevantAsnPosition || config.GlobalConf.RelevantAsnPosition == 0 {
		return "0"
	}
	b := make([]byte, 0)
	for i := range asPath.Asn[0:config.GlobalConf.RelevantAsnPosition] {
		b = strconv.AppendInt(b, int64(asPath.Asn[i]), 10)
		b = append(b, ',')
	}
	return string(b)
}

func pathToString(asPath common.AsPathList) string {
	b := make([]byte, 0)
	for i := range asPath.Asn {
		b = strconv.AppendInt(b, int64(asPath.Asn[i]), 10)
		b = append(b, ',')
	}
	return string(b)
}

func pathsEqual(path1, path2 common.AsPathList) bool {
	if len(path1.Asn) != len(path2.Asn) {
		return false
	}
	for i := range path1.Asn {
		if path1.Asn[i] != path2.Asn[i] {
			return false
		}
	}
	return true
}

func incrementUint64(n uint64) uint64 {
	if n == math.MaxUint64 {
		return n
	}
	return n + 1
}

func getActiveFlapList() []*Flap {
	aFlap := make([]*Flap, 0)
	activeFlapListMu.RLock()
	for i := range activeFlapList {
		activeFlapList[i].RLock()
		if !activeFlapList[i].meetsMinimumAge {
			activeFlapList[i].RUnlock()
			continue
		}
		activeFlapList[i].RUnlock()
		aFlap = append(aFlap, activeFlapList[i])
	}
	activeFlapListMu.RUnlock()
	return aFlap
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

	statSubscribers     = make([]chan statistic, 0)
	statSubscribersLock sync.Mutex
)

func addStatSubscriber() chan statistic {
	statSubscribersLock.Lock()
	defer statSubscribersLock.Unlock()
	c := make(chan statistic, 2)
	statSubscribers = append(statSubscribers, c)
	return c
}

func statTracker() {
	for {
		time.Sleep(5 * time.Second)
		activeFlapListMu.RLock()
		flapListLength := len(activeFlapList)
		activeFlapListMu.RUnlock()

		newStatistic := statistic{
			Time:          time.Now().Unix(),
			Changes:       globalTotalRouteChangeCounter.Swap(0),
			ListedChanges: globalListedRouteChangeCounter.Swap(0),
			Active:        flapListLength,
		}

		statSubscribersLock.Lock()
		for _, subscriber := range statSubscribers {
			select {
			case subscriber <- newStatistic:
			default:
			}
		}
		statSubscribersLock.Unlock()

		statListLock.Lock()
		statList = append(statList, newStatistic)
		if len(statList) > 60 {
			statList = statList[1:]
		}
		statListLock.Unlock()
	}
}
