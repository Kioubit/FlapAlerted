package monitor

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/config"
	"container/list"
	"math"
	"strconv"
	"strings"
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
	AnnouncementCount uint64
	WithdrawalCount   uint64
}

type pathEntry struct {
	key   string
	info  *PathInfo
	ticks int
}

func (pt *PathTracker) Record(path common.AsPath, isWithdrawal bool) {
	if !config.GlobalConf.KeepPathInfo {
		return
	}
	pt.lock.Lock()
	defer pt.lock.Unlock()

	key := pathToString(path)

	if elem, exists := pt.paths[key]; exists {
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
		key:  key,
		info: pathInfoEntry,
	})
	pt.paths[key] = elem
}
func pathToString(asPath common.AsPath) string {
	if len(asPath) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(asPath) * 11) // Pre-allocate

	for i, as := range asPath {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatUint(uint64(as), 10))
	}

	return sb.String()
}

func (pt *PathTracker) Get(key string) *PathInfo {
	pt.lock.RLock()
	defer pt.lock.RUnlock()

	if elem, exists := pt.paths[key]; exists {
		return elem.Value.(*pathEntry).info
	} else {
		return nil
	}
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
			entryCount := SafeAddUint64(entry.info.AnnouncementCount, entry.info.WithdrawalCount)

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

func (pt *PathTracker) All() func(func(string, *PathInfo) bool) {
	return func(yield func(string, *PathInfo) bool) {
		pt.lock.RLock()
		defer pt.lock.RUnlock()
		for elem := pt.order.Front(); elem != nil; elem = elem.Next() {
			entry := elem.Value.(*pathEntry)
			if !yield(entry.key, entry.info) {
				return
			}
		}
	}
}

func NewPathTracker(limit int) *PathTracker {
	return &PathTracker{
		paths: make(map[string]*list.Element),
		order: list.New(),
		limit: limit,
	}
}
