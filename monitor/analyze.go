package monitor

import (
	"FlapAlertedPro/bgp"
	"fmt"
	"math"
	"net"
	"strconv"
	"sync"
	"time"
)

func StartMonitoring(asn uint32, flapPeriod int64, notifytarget uint64, addpath bool, perPeerState bool, debug bool, notifyOnce bool) {
	FlapPeriod = flapPeriod
	NotifyTarget = notifytarget
	updateChannel := make(chan *bgp.UserUpdate, 10000)
	if addpath {
		bgp.GlobalAdpath = true
	}
	if perPeerState {
		GlobalPerPeerState = true
	}
	if debug {
		bgp.GlobalDebug = true
	}
	if notifyOnce {
		GlobalNotifyOnce = true
	}

	go bgp.StartBGP(asn, updateChannel)
	go cleanUpFlapList()
	go moduleCallback()
	processUpdates(updateChannel)
}

var GlobalPerPeerState = false
var GlobalNotifyOnce = false

type Flap struct {
	Cidr                 string
	Paths                []bgp.AsPath
	LastPath             map[uint32]bgp.AsPath
	PathChangeCount      uint64
	PathChangeCountTotal uint64
	FirstSeen            int64
	LastSeen             int64
}

var (
	flapList   []*Flap
	flaplistMu sync.RWMutex
)

var (
	FlapPeriod   int64  = 2
	NotifyTarget uint64 = 10
)

func processUpdates(updateChannel chan *bgp.UserUpdate) {
	for {
		update := <-updateChannel
		if update == nil {
			return
		}

		for len(updateChannel) > 9800 {
			fmt.Println("[WARNING] Can't keep up! Dropping 100 updates")
			for i := 0; i < 100; i++ {
				<-updateChannel
			}
		}

		flaplistMu.Lock()
		for i := range update.Prefix {
			if update.Prefix[i].Prefix4 != nil {
				if len(update.Prefix[i].Prefix4) == 0 {
					continue
				}
				updateList(update.Prefix[i].Prefix4, update.Prefix[i].PrefixLenBits, update.Path, false)
			}
			if update.Prefix[i].Prefix6 != nil {
				if len(update.Prefix[i].Prefix6) == 0 {
					continue
				}
				updateList(update.Prefix[i].Prefix6, update.Prefix[i].PrefixLenBits, update.Path, true)
			}
		}
		flaplistMu.Unlock()

	}

}

func cleanUpFlapList() {
	for {
		select {
		case <-time.After(2 * time.Duration(FlapPeriod) * time.Second):
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

func updateList(prefix []byte, prefixlenBits int, aspath []bgp.AsPath, isV6 bool) {
	if len(aspath) == 0 {
		return
	}
	cleanPath := aspath[0] // Multiple AS paths in a single update message currently unsupported (not used by bird)

	cidr := toNetCidr(prefix, prefixlenBits, isV6)

	currentTime := time.Now().Unix()
	var found = false
	for i := range flapList {
		if cidr == flapList[i].Cidr {
			found = true
			if flapList[i].LastSeen+FlapPeriod <= currentTime {
				flapList[i].PathChangeCount = 0
				flapList[i].Paths = []bgp.AsPath{cleanPath}
				flapList[i].LastPath[getFirstAsn(cleanPath)] = cleanPath
			} else {
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
				if flapList[i].PathChangeCount > NotifyTarget {
					if GlobalNotifyOnce {
						if flapList[i].PathChangeCount > NotifyTarget+1 {
							return
						}
					}
					flapList[i].PathChangeCount = 0
					go mainNotify(flapList[i])
				}
			}
			break
		}
	}

	if !found {
		newFlap := &Flap{
			Cidr:                 cidr,
			LastSeen:             currentTime,
			FirstSeen:            currentTime,
			PathChangeCount:      1,
			PathChangeCountTotal: 1,
			Paths:                []bgp.AsPath{cleanPath},
			LastPath:             make(map[uint32]bgp.AsPath),
		}
		newFlap.LastPath[getFirstAsn(cleanPath)] = cleanPath
		flapList = append(flapList, newFlap)
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

func toNetCidr(prefix []byte, prefixlenBits int, isV6 bool) string {
	if isV6 {
		needBytes := 16 - len(prefix)
		toAppend := make([]byte, needBytes)
		prefix = append(prefix, toAppend...)
		ip := net.IP{prefix[0], prefix[1], prefix[2], prefix[3], prefix[4], prefix[5],
			prefix[6], prefix[7], prefix[8], prefix[9], prefix[10], prefix[11], prefix[12],
			prefix[13], prefix[14], prefix[15]}
		cidr := ip.String() + "/" + strconv.Itoa(prefixlenBits)
		return cidr
	} else {
		needBytes := 4 - len(prefix)
		toAppend := make([]byte, needBytes)
		prefix = append(prefix, toAppend...)
		ip := net.IP{prefix[0], prefix[1], prefix[2], prefix[3]}
		cidr := ip.String() + "/" + strconv.Itoa(prefixlenBits)
		return cidr
	}
}

func incrementUint64(n uint64) uint64 {
	if n == math.MaxUint64-1 {
		return n
	}
	return n + 1
}
