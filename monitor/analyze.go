//go:build !core_doubleAddPath
// +build !core_doubleAddPath

package monitor

import (
	"FlapAlertedPro/bgp"
	"log"
	"math"
	"sync"
	"time"
)

var (
	GlobalPerPeerState bool   = false
	GlobalNotifyOnce   bool   = false
	GlobalKeepPathInfo bool   = true
	FlapPeriod         int64  = 2
	NotifyTarget       uint64 = 10
)

type Flap struct {
	Cidr                 string
	LastPath             map[uint32]bgp.AsPath
	Paths                []bgp.AsPath
	PathChangeCount      uint64
	PathChangeCountTotal uint64
	FirstSeen            int64
	LastSeen             int64
}

func StartMonitoring(asn uint32, flapPeriod int64, notifytarget uint64, addpath bool, perPeerState bool, debug bool, notifyOnce bool, keepPathInfo bool) {
	FlapPeriod = flapPeriod
	NotifyTarget = notifytarget
	bgp.GlobalAddpath = addpath
	GlobalPerPeerState = perPeerState
	bgp.GlobalDebug = debug
	GlobalNotifyOnce = notifyOnce
	GlobalKeepPathInfo = keepPathInfo

	updateChannel := make(chan *bgp.UserUpdate, 11000)
	// Initialize activeFlapList and flapMap
	flapMap = make(map[string]*Flap)
	activeFlapList = make([]*Flap, 0)

	go bgp.StartBGP(asn, updateChannel)
	go cleanUpFlapList()
	go moduleCallback()
	processUpdates(updateChannel)
}

var (
	flapMap          map[string]*Flap
	flapMapMu        sync.Mutex
	activeFlapList   []*Flap
	activeFlapListMu sync.RWMutex
)

func processUpdates(updateChannel chan *bgp.UserUpdate) {
	for {
		update := <-updateChannel
		if update == nil {
			return
		}

		if len(updateChannel) > 10700 {
			go updateDropper(updateChannel)

		}

		flapMapMu.Lock()
		for i := range update.Prefix {
			updateList(update.Prefix[i], update.Path)
		}
		flapMapMu.Unlock()
	}
}

func cleanUpFlapList() {
	for {
		select {
		case <-time.After(1 * time.Duration(FlapPeriod) * time.Second):
		}

		currentTime := time.Now().Unix()
		flapMapMu.Lock()
		activeFlapListMu.Lock()
		for key, element := range flapMap {
			if element.LastSeen+FlapPeriod <= currentTime {
				delete(flapMap, key)
			}

			var activeIndex int
			var found = false
			for i := range activeFlapList {
				if activeFlapList[i].Cidr == element.Cidr {
					activeIndex = i
					found = true
				}
			}
			if found {
				activeFlapList[activeIndex] = activeFlapList[len(activeFlapList)-1]
				activeFlapList = activeFlapList[:len(activeFlapList)-1]
			}

		}
		flapMapMu.Unlock()
		activeFlapListMu.Unlock()
	}
}

func updateList(cidr string, aspath []bgp.AsPath) {
	if len(aspath) == 0 {
		return
	}
	cleanPath := aspath[0] // Multiple AS paths in a single update message currently unsupported (not used by bird)

	currentTime := time.Now().Unix()
	obj := flapMap[cidr]

	if obj == nil {
		newFlap := &Flap{
			Cidr:                 cidr,
			LastSeen:             currentTime,
			FirstSeen:            currentTime,
			PathChangeCount:      1,
			PathChangeCountTotal: 1,
			LastPath:             make(map[uint32]bgp.AsPath),
		}
		if GlobalKeepPathInfo {
			newFlap.Paths = []bgp.AsPath{cleanPath}
		}
		newFlap.LastPath[getFirstAsn(cleanPath)] = cleanPath
		flapMap[cidr] = newFlap
		return
	}

	// If the entry already exists

	if GlobalKeepPathInfo {
		exists := false
		for b := range flapMap[cidr].Paths {
			if pathsEqual(flapMap[cidr].Paths[b], cleanPath) {
				exists = true
				break
			}
		}
		if !exists {
			flapMap[cidr].Paths = append(flapMap[cidr].Paths, cleanPath)
		}
	}

	if !pathsEqual(flapMap[cidr].LastPath[getFirstAsn(cleanPath)], cleanPath) {
		if GlobalPerPeerState {
			if len(flapMap[cidr].LastPath[getFirstAsn(cleanPath)].Asn) == 0 {
				flapMap[cidr].LastPath[getFirstAsn(cleanPath)] = cleanPath
				return
			}
		}

		flapMap[cidr].PathChangeCount = incrementUint64(flapMap[cidr].PathChangeCount)
		flapMap[cidr].PathChangeCountTotal = incrementUint64(flapMap[cidr].PathChangeCountTotal)

		flapMap[cidr].LastSeen = currentTime
		flapMap[cidr].LastPath[getFirstAsn(cleanPath)] = cleanPath
		if flapMap[cidr].PathChangeCount >= NotifyTarget {
			flapMap[cidr].PathChangeCount = 0
			if GlobalNotifyOnce {
				if flapMap[cidr].PathChangeCountTotal > NotifyTarget {
					return
				}
			}
			if flapMap[cidr].PathChangeCountTotal == NotifyTarget {
				activeFlapListMu.Lock()
				activeFlapList = append(activeFlapList, flapMap[cidr])
				activeFlapListMu.Unlock()
			}
			go mainNotify(flapMap[cidr])
		}
	}
}

func getFirstAsn(aspath bgp.AsPath) uint32 {
	if GlobalPerPeerState {
		if len(aspath.Asn) == 0 {
			return 0
		}
		return aspath.Asn[0]
	} else {
		return 0
	}
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

func updateDropper(updateChannel chan *bgp.UserUpdate) {
	log.Println("[WARNING] Can't keep up! Dropping some updates")
	for len(updateChannel) > 10700 {
		for i := 0; i < 50; i++ {
			<-updateChannel
		}
	}
	log.Println("[INFO] Recovered")
}
