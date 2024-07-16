package monitor

import (
	"FlapAlerted/bgp"
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/update"
	"FlapAlerted/config"
	"encoding/json"
	"fmt"
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
	PathChangeCount      uint64
	PathChangeCountTotal uint64
	FirstSeen            int64
	LastSeen             int64
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

	updateChannel := make(chan update.Msg, 100)
	// Initialize activeFlapList and flapMap
	flapMap = make(map[string]*Flap)
	activeFlapList = make([]*Flap, 0)

	go processUpdates(updateChannel)
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

var globalRouteChangeCounter atomic.Uint64

func processUpdates(updateChannel chan update.Msg) {
	for {
		u, ok := <-updateChannel
		if !ok {
			return
		}

		if config.GlobalConf.Debug {
			k, err := json.Marshal(u)
			fmt.Println(string(k), err)
		}

		nlri, foundNlri, err := u.GetMpReachNLRI()
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
		if foundNlri {
			for i := range nlri.NLRI {
				if config.GlobalConf.Debug {
					fmt.Println("UPDATE", nlri.NLRI[i].ToNetCidr().String(), asPath)
				}
				updateList(nlri.NLRI[i].ToNetCidr(), asPath)
			}
		}
		for i := range u.NetworkLayerReachabilityInformation {
			updateList(u.NetworkLayerReachabilityInformation[i].ToNetCidr(), asPath)
			if config.GlobalConf.Debug {
				fmt.Println("UPDATE", u.NetworkLayerReachabilityInformation[i].ToNetCidr().String(), asPath)
			}
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

func updateList(prefix netip.Prefix, asPath []common.AsPathList) {
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
			PathChangeCount:      0,
			PathChangeCountTotal: 0,
			LastPath:             make(map[string]common.AsPathList),
			Paths:                make(map[string]*PathInfo),
		}
		if config.GlobalConf.KeepPathInfo {
			newFlap.Paths[pathToString(cleanPath)] = &PathInfo{Path: cleanPath, Count: 1}
		}
		newFlap.LastPath[getRelevantASN(cleanPath)] = cleanPath
		flapMap[cidr] = newFlap

		globalRouteChangeCounter.Add(1)

		// Notify for every Update
		if config.GlobalConf.RouteChangeCounter == 0 {
			newFlap.PathChangeCountTotal = 1
			newFlap.PathChangeCount = 1
			activeFlapListMu.Lock()
			activeFlapList = append(activeFlapList, newFlap)
			activeFlapListMu.Unlock()
			go mainNotify(newFlap)
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

		obj.PathChangeCount = incrementUint64(obj.PathChangeCount)
		obj.PathChangeCountTotal = incrementUint64(obj.PathChangeCountTotal)
		globalRouteChangeCounter.Add(1)

		obj.LastSeen = currentTime
		obj.LastPath[getRelevantASN(cleanPath)] = cleanPath

		if config.GlobalConf.RouteChangeCounter != 0 {
			if obj.PathChangeCount == uint64(config.GlobalConf.RouteChangeCounter) {
				if obj.PathChangeCountTotal == uint64(config.GlobalConf.RouteChangeCounter) {
					activeFlapListMu.Lock()
					activeFlapList = append(activeFlapList, obj)
					activeFlapListMu.Unlock()
				}
				obj.PathChangeCount = 0
				go mainNotify(obj)
			}
		}
	}
}

func getRelevantASN(asPath common.AsPathList) string {
	pathLen := len(asPath.Asn)

	if pathLen < int(config.GlobalConf.RelevantAsnPosition) || config.GlobalConf.RelevantAsnPosition == 0 {
		return "0"
	}
	b := make([]byte, 0, pathLen*11)
	for i := range asPath.Asn[0:config.GlobalConf.RelevantAsnPosition] {
		b = strconv.AppendInt(b, int64(asPath.Asn[i]), 10)
		b = append(b, ',')
	}
	return string(b)
}

func pathToString(asPath common.AsPathList) string {
	b := make([]byte, 0, len(asPath.Asn)*11)
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
		aFlap = append(aFlap, activeFlapList[i])
	}
	activeFlapListMu.RUnlock()
	return aFlap
}

type statistic struct {
	Time    int64
	Changes uint64
	Active  int
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
			Time:    time.Now().Unix(),
			Changes: globalRouteChangeCounter.Swap(0),
			Active:  flapListLength,
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
