package monitor

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/config"
	"container/list"
	"encoding/binary"
	"math"
	"sync"
)

type PathTracker struct {
	paths map[string]*list.Element
	order *list.List
	limit int
	lock  sync.RWMutex
}

type PathInfo struct {
	Path              common.AsPath
	AnnouncementCount uint64 `json:"ac"`
	WithdrawalCount   uint64 `json:"wc"`
}

type pathEntry struct {
	key   string
	info  *PathInfo
	ticks int
}

func (pt *PathTracker) record(path common.AsPath, isWithdrawal bool) {
	if !config.GlobalConf.KeepPathInfo {
		return
	}
	key := pathToKey(path)

	pt.lock.Lock()
	defer pt.lock.Unlock()

	if elem, exists := pt.paths[string(key)]; exists {
		entry := elem.Value.(*pathEntry)
		if isWithdrawal {
			incrementUint64(&entry.info.WithdrawalCount)
		} else {
			incrementUint64(&entry.info.AnnouncementCount)
		}

		entry.ticks = 0
		pt.order.MoveToBack(elem)
		return
	}

	if len(pt.paths) >= pt.limit {
		pt.deleteLeastValuable()
	}

	pathInfoEntry := &PathInfo{
		Path: path,
	}
	if isWithdrawal {
		pathInfoEntry.WithdrawalCount = 1
	} else {
		pathInfoEntry.AnnouncementCount = 1
	}

	elem := pt.order.PushBack(&pathEntry{
		key:  string(key),
		info: pathInfoEntry,
	})
	pt.paths[string(key)] = elem
}
func pathToKey(p common.AsPath) []byte {
	if len(p) == 0 {
		return nil
	}
	buf := make([]byte, len(p)*4)
	for i, v := range p {
		binary.LittleEndian.PutUint32(buf[i*4:], v)
	}
	return buf
}

func (pt *PathTracker) deleteLeastValuable() {
	if pt.order.Len() == 0 {
		return
	}

	var toDelete = pt.order.Front()
	if toDelete == nil {
		return
	}

	var minCount uint64 = math.MaxUint64

	// Oldest element
	toDelete.Value.(*pathEntry).ticks++
	if toDelete.Value.(*pathEntry).ticks < 10000 {
		steps := 0
		for elem := pt.order.Front(); elem != nil && steps < 50; elem = elem.Next() {
			entry := elem.Value.(*pathEntry)
			entryCount := safeAddUint64(entry.info.AnnouncementCount, entry.info.WithdrawalCount)

			if entryCount < minCount {
				minCount = entryCount
				toDelete = elem
				if entryCount == 1 {
					break
				}
			}
			steps++
		}
	}

	entry := pt.order.Remove(toDelete).(*pathEntry)
	delete(pt.paths, entry.key)
}

func (pt *PathTracker) All() func(func(*PathInfo) bool) {
	return func(yield func(*PathInfo) bool) {
		pt.lock.RLock()
		defer pt.lock.RUnlock()
		for elem := pt.order.Front(); elem != nil; elem = elem.Next() {
			entry := elem.Value.(*pathEntry)
			if !yield(entry.info) {
				return
			}
		}
	}
}

func newPathTracker(limit int) *PathTracker {
	return &PathTracker{
		paths: make(map[string]*list.Element),
		order: list.New(),
		limit: limit,
	}
}
