//go:build !core_doubleAddPath
// +build !core_doubleAddPath

package monitor

import (
	"FlapAlertedPro/bgp"
	"log"
	"math"
	"sync"
	"sync/atomic"
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
	go bgp.StartBGP(asn, updateChannel)
	go cleanUpFlapList()
	go moduleCallback()
	processUpdates(updateChannel)
}

var (
	flapList   []*Flap
	flaplistMu sync.RWMutex
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

		flaplistMu.Lock()
		for i := range update.Prefix {
			updateList(update.Prefix[i], update.Path)
		}
		flaplistMu.Unlock()
	}
}

func cleanUpFlapList() {
	for {
		select {
		case <-time.After(1 * time.Duration(FlapPeriod) * time.Second):
		}

		currentTime := time.Now().Unix()
		newFlapList := make([]*Flap, 0)
		flaplistMu.Lock()
		for i := range flapList {
			if flapList[i].LastSeen+FlapPeriod <= currentTime {
				continue
			}
			newFlapList = append(newFlapList, flapList[i])
		}
		flapList = newFlapList
		flaplistMu.Unlock()
	}
}

func updateList(cidr string, aspath []bgp.AsPath) {
	if len(aspath) == 0 {
		return
	}
	cleanPath := aspath[0] // Multiple AS paths in a single update message currently unsupported (not used by bird)

	currentTime := time.Now().Unix()
	for i := range flapList {
		if cidr == flapList[i].Cidr {
			if GlobalKeepPathInfo {
				exists := false
				for b := range flapList[i].Paths {
					if pathsEqual(flapList[i].Paths[b], cleanPath) {
						exists = true
						break
					}
				}
				if !exists {
					flapList[i].Paths = append(flapList[i].Paths, cleanPath)
				}
			}

			if !pathsEqual(flapList[i].LastPath[getFirstAsn(cleanPath)], cleanPath) {

				if GlobalPerPeerState {
					if len(flapList[i].LastPath[getFirstAsn(cleanPath)].Asn) == 0 {
						flapList[i].LastPath[getFirstAsn(cleanPath)] = cleanPath
						return
					}
				}

				flapList[i].PathChangeCount = incrementUint64(flapList[i].PathChangeCount)
				flapList[i].PathChangeCountTotal = incrementUint64(flapList[i].PathChangeCountTotal)

				flapList[i].LastSeen = currentTime
				flapList[i].LastPath[getFirstAsn(cleanPath)] = cleanPath
				if flapList[i].PathChangeCount >= NotifyTarget {
					flapList[i].PathChangeCount = 0
					if GlobalNotifyOnce {
						if flapList[i].PathChangeCount >= NotifyTarget+1 {
							return
						}
					}
					go mainNotify(flapList[i])
				}
			}
			return
		}
	}

	// If not returned above
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
	flapList = append(flapList, newFlap)
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

var updateDropperRunning int32 = 0

func updateDropper(updateChannel chan *bgp.UserUpdate) {
	log.Println("[WARNING] Can't keep up! Dropping some updates")
	for len(updateChannel) > 10700 {
		for i := 0; i < 50; i++ {
			<-updateChannel
		}
	}
	log.Println("[INFO] Recovered")
	atomic.StoreInt32(&updateDropperRunning, int32(0))
}
