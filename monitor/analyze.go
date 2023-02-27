package monitor

import (
	"FlapAlertedPro/bgp"
	"FlapAlertedPro/config"
	"log"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const PathLimit = 1000

type Flap struct {
	Cidr                 string
	LastPath             map[string]bgp.AsPath
	Paths                map[string]*PathInfo
	PathChangeCount      uint64
	PathChangeCountTotal uint64
	FirstSeen            int64
	LastSeen             int64
}

type PathInfo struct {
	Path  bgp.AsPath
	Count uint64
}

func StartMonitoring(conf config.UserConfig) {
	config.GlobalConf = conf
	if config.GlobalConf.RelevantAsnPosition < 0 {
		log.Fatal("Invalid RelevantAsnPosition value")
	}

	updateChannel := make(chan *bgp.UserUpdate, 11000)
	// Initialize activeFlapList and flapMap
	flapMap = make(map[string]*Flap)
	activeFlapList = make([]*Flap, 0)

	go processUpdates(updateChannel)
	go moduleCallback()
	go bgp.StartBGP(updateChannel)
	cleanUpFlapList()
}

var (
	flapMap          map[string]*Flap
	flapMapMu        sync.RWMutex
	activeFlapList   []*Flap
	activeFlapListMu sync.RWMutex
	currentTime      int64
)

func processUpdates(updateChannel chan *bgp.UserUpdate) {
	for {
		update := <-updateChannel
		if update == nil {
			return
		}

		if len(updateChannel) > 10700 {
			if atomic.CompareAndSwapInt32(&updateDropperRunning, int32(0), int32(1)) {
				go updateDropper(updateChannel)
			}
		}

		flapMapMu.Lock()
		currentTime = time.Now().Unix()
		for i := range update.Prefix {
			updateList(update.Prefix[i], update.Path)
		}
		flapMapMu.Unlock()
	}
}

func cleanUpFlapList() {
	for {
		select {
		case <-time.After(1 * time.Duration(config.GlobalConf.FlapPeriod) * time.Second):
		}

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

func updateList(cidr string, asPath []bgp.AsPath) {
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
			LastPath:             make(map[string]bgp.AsPath),
			Paths:                make(map[string]*PathInfo),
		}
		if config.GlobalConf.KeepPathInfo {
			newFlap.Paths[pathToString(cleanPath)] = &PathInfo{Path: cleanPath, Count: 1}
		}
		newFlap.LastPath[getRelevantASN(cleanPath)] = cleanPath
		flapMap[cidr] = newFlap

		// Notify for every Update
		if config.GlobalConf.RouteChangeCounter == 0 {
			newFlap.PathChangeCountTotal = 1
			activeFlapListMu.Lock()
			activeFlapList = append(activeFlapList, newFlap)
			activeFlapListMu.Unlock()
			go mainNotify(newFlap)
		}
		return
	}

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

		obj.LastSeen = currentTime
		obj.LastPath[getRelevantASN(cleanPath)] = cleanPath

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

func getRelevantASN(asPath bgp.AsPath) string {
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

func pathToString(asPath bgp.AsPath) string {
	b := make([]byte, 0, len(asPath.Asn)*11)
	for i := range asPath.Asn {
		b = strconv.AppendInt(b, int64(asPath.Asn[i]), 10)
		b = append(b, ',')
	}
	return string(b)
}

func pathsEqual(path1, path2 bgp.AsPath) bool {
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

var updateDropperRunning int32 = 0

func updateDropper(updateChannel chan *bgp.UserUpdate) {
	for len(updateChannel) > 10700 {
		for i := 0; i < 50; i++ {
			<-updateChannel
		}
	}
	atomic.StoreInt32(&updateDropperRunning, int32(0))
}

func getActiveFlapList() []Flap {
	aFlap := make([]Flap, 0)
	activeFlapListMu.RLock()
	for i := range activeFlapList {
		aFlap = append(aFlap, *activeFlapList[i])
	}
	activeFlapListMu.RUnlock()
	return aFlap
}
