package table

import (
	"FlapAlerted/bgp/common"
	"net/netip"
)

type PrefixTable struct {
	table          map[netip.Prefix]*Entry
	pathChangeChan chan PathChange
}

func NewPrefixTable(pathChangeChan chan PathChange) *PrefixTable {
	return &PrefixTable{table: make(map[netip.Prefix]*Entry), pathChangeChan: pathChangeChan}
}

type PathChange struct {
	Prefix       netip.Prefix
	IsWithdrawal bool
	OldPath      common.AsPath
}

type Entry struct {
	Paths map[uint32]common.AsPath
}

func (t *PrefixTable) update(prefix netip.Prefix, pathID uint32, isWithdrawal bool, asPath common.AsPath) {
	if isWithdrawal {
		if entry, ok := t.table[prefix]; ok {
			if oldPath, exists := entry.Paths[pathID]; exists {
				t.pathChangeChan <- PathChange{
					Prefix:       prefix,
					IsWithdrawal: true,
					OldPath:      oldPath,
				}
				delete(entry.Paths, pathID)
				if len(entry.Paths) == 0 {
					delete(t.table, prefix)
				}
			}
		}
	} else {
		entry, found := t.table[prefix]
		if !found {
			entry = &Entry{Paths: make(map[uint32]common.AsPath)}
			t.table[prefix] = entry
		} else {
			if oldPath, existed := entry.Paths[pathID]; existed {
				t.pathChangeChan <- PathChange{
					Prefix:       prefix,
					IsWithdrawal: false,
					OldPath:      oldPath,
				}
			}
		}
		entry.Paths[pathID] = asPath
	}
}
