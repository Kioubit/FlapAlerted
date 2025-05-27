package monitor

import (
	"container/list"
	"math"
)

type PathTracker struct {
	paths map[string]*list.Element
	order *list.List
}

type pathEntry struct {
	key   string
	info  *PathInfo
	ticks int
}

func (pt *PathTracker) Update(key string) {
	if elem, exists := pt.paths[key]; exists {
		pt.order.MoveToBack(elem)
		elem.Value.(*pathEntry).ticks = 0
	}
}

func (pt *PathTracker) Set(key string, info *PathInfo) {
	if elem, exists := pt.paths[key]; exists {
		pt.order.MoveToBack(elem)
		elem.Value.(*pathEntry).info = info
		elem.Value.(*pathEntry).ticks = 0
		return
	}

	elem := pt.order.PushBack(&pathEntry{key: key, info: info})
	pt.paths[key] = elem
}

func (pt *PathTracker) Get(key string) *PathInfo {
	if elem, exists := pt.paths[key]; exists {
		return elem.Value.(*pathEntry).info
	} else {
		return nil
	}
}

func (pt *PathTracker) Length() int {
	return len(pt.paths)
}

func (pt *PathTracker) DeleteLeastValuable() {
	if pt.order.Len() == 0 {
		return
	}

	var toDelete = pt.order.Front()
	var minCount uint64 = math.MaxUint64

	// Oldest element
	toDelete.Value.(*pathEntry).ticks++
	if toDelete.Value.(*pathEntry).ticks < 10000 {
		steps := 0
		for elem := pt.order.Front(); elem != nil && steps < 50; elem = elem.Next() {
			entry := elem.Value.(*pathEntry)
			entryCount := entry.info.Count

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
		for elem := pt.order.Front(); elem != nil; elem = elem.Next() {
			entry := elem.Value.(*pathEntry)
			if !yield(entry.key, entry.info) {
				return
			}
		}
	}
}

func NewPathTracker() *PathTracker {
	return &PathTracker{
		paths: make(map[string]*list.Element),
		order: list.New(),
	}
}
