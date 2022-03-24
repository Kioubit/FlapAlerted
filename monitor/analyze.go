package monitor

import (
	"FlapAlertedPro/bgp"
	"errors"
	"math"
	"net"
	"strconv"
	"sync"
	"time"
)

func StartMonitoring(asn uint32, flapPeriod int64, notifytarget uint64) {
	FlapPeriod = flapPeriod
	NotifyTarget = notifytarget
	updateChannel := make(chan *bgp.UserUpdate, 200)
	bgp.GlobalDebug = false
	go bgp.StartBGP(asn, updateChannel)
	go cleanUpFlapList()
	go moduleCallback()
	processUpdates(updateChannel)
}

type Flap struct {
	Cidr                 string
	Paths                []bgp.AsPath
	LastPath             bgp.AsPath
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

	}

}

func cleanUpFlapList() {
	for {
		select {
		case <-time.After(2 * time.Duration(FlapPeriod) * time.Second):
		}
		flaplistMu.Lock()
		newFlapList := make([]*Flap, 0)
		for i := range flapList {
			if flapList[i].LastSeen+FlapPeriod <= time.Now().Unix() {
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
	cleanPath := aspath[0] // Multiple AS paths currently unsupported

	cidr, err := toNetCidr(prefix, prefixlenBits, isV6)
	if err != nil {
		return
	}

	flaplistMu.Lock()
	defer flaplistMu.Unlock()

	var found = false
	for i := range flapList {
		if cidr.String() == flapList[i].Cidr {
			found = true
			if flapList[i].LastSeen+FlapPeriod <= time.Now().Unix() {
				flapList[i].PathChangeCount = 0
				flapList[i].Paths = []bgp.AsPath{cleanPath}
				flapList[i].LastPath = cleanPath
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
			if !pathsEqual(flapList[i].LastPath, cleanPath) {
				flapList[i].PathChangeCount = incrementUint64(flapList[i].PathChangeCount)
				flapList[i].PathChangeCountTotal = incrementUint64(flapList[i].PathChangeCountTotal)

				flapList[i].LastSeen = time.Now().Unix()
				flapList[i].LastPath = cleanPath
				if flapList[i].PathChangeCount > NotifyTarget {
					flapList[i].PathChangeCount = 0
					go mainNotify(flapList[i])
				}
			}
			break
		}
	}

	if !found {
		newFlap := &Flap{
			Cidr:                 cidr.String(),
			LastSeen:             time.Now().Unix(),
			FirstSeen:            time.Now().Unix(),
			PathChangeCount:      1,
			PathChangeCountTotal: 1,
			Paths:                []bgp.AsPath{cleanPath},
			LastPath:             cleanPath,
		}
		flapList = append(flapList, newFlap)
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

func toNetCidr(prefix []byte, prefixlenBits int, isV6 bool) (*net.IPNet, error) {
	if isV6 {
		needBytes := 16 - len(prefix)
		toAppend := make([]byte, needBytes)
		prefix = append(prefix, toAppend...)
		ip := net.IP{prefix[0], prefix[1], prefix[2], prefix[3], prefix[4], prefix[5],
			prefix[6], prefix[7], prefix[8], prefix[9], prefix[10], prefix[11], prefix[12],
			prefix[13], prefix[14], prefix[15]}
		_, cidr, err := net.ParseCIDR(ip.String() + "/" + strconv.Itoa(prefixlenBits))
		if err != nil {
			return nil, errors.New("error parsing")
		}
		return cidr, nil
	} else {
		needBytes := 4 - len(prefix)
		toAppend := make([]byte, needBytes)
		prefix = append(prefix, toAppend...)
		ip := net.IP{prefix[0], prefix[1], prefix[2], prefix[3]}
		_, cidr, err := net.ParseCIDR(ip.String() + "/" + strconv.Itoa(prefixlenBits))
		if err != nil {
			return nil, errors.New("error parsing")
		}
		return cidr, nil
	}
}

func incrementUint64(n uint64) uint64 {
	if n == math.MaxUint64-1 {
		return n
	}
	return n + 1
}
