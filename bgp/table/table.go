package table

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/notification"
	"FlapAlerted/config"
	"context"
	"net/netip"
	"sync/atomic"
)

type PrefixTable struct {
	table               map[netip.Prefix]*Entry
	pathChangeChan      chan PathChange
	importCount         atomic.Uint32
	sessionCancellation context.CancelCauseFunc
}

func NewPrefixTable(pathChangeChan chan PathChange, sessionCancellation context.CancelCauseFunc) *PrefixTable {
	return &PrefixTable{table: make(map[netip.Prefix]*Entry), pathChangeChan: pathChangeChan, sessionCancellation: sessionCancellation}
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
				t.importCount.Add(^uint32(0))
				delete(entry.Paths, pathID)
				if len(entry.Paths) == 0 {
					delete(t.table, prefix)
				}
			}
		}
	} else {
		entry, found := t.table[prefix]
		if !found {
			t.importCount.Add(1)
			entry = &Entry{Paths: make(map[uint32]common.AsPath)}
			t.table[prefix] = entry
		} else {
			if oldPath, existed := entry.Paths[pathID]; existed {
				t.pathChangeChan <- PathChange{
					Prefix:       prefix,
					IsWithdrawal: false,
					OldPath:      oldPath,
				}
			} else {
				t.importCount.Add(1)
			}
		}
		entry.Paths[pathID] = asPath
		if t.importCount.Load() > config.GlobalConf.ImportLimit {
			t.sessionCancellation(notification.ErrImportLimit)
		}
	}
}

func (t *PrefixTable) ImportCount() uint32 {
	return t.importCount.Load()
}
