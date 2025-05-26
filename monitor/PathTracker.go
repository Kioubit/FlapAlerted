package monitor

import "container/list"

type PathTracker struct {
	paths map[string]*list.Element
	order *list.List
}

type pathEntry struct {
	key  string
	info *PathInfo
}

func (pt *PathTracker) Set(key string, info *PathInfo) {
	if elem, exists := pt.paths[key]; exists {
		pt.order.MoveToBack(elem)
		elem.Value.(*pathEntry).info = info
		return
	}

	elem := pt.order.PushBack(&pathEntry{key: key, info: info})
	pt.paths[key] = elem
}

func (pt *PathTracker) Get(key string) *PathInfo {
	return pt.paths[key].Value.(*PathInfo)
}

func (pt *PathTracker) Length() int {
	return len(pt.paths)
}

func (pt *PathTracker) DeleteOldest() {
	if pt.order.Len() == 0 {
		return
	}
	front := pt.order.Front()
	entry := pt.order.Remove(front).(*pathEntry)
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
