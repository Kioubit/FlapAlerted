package table

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/notification"
	"FlapAlerted/config"
	"context"
	"net/netip"
)

type PrefixTable struct {
	table               map[netip.Prefix]*Entry
	pathChangeChan      chan PathChange
	importCount         int
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
				t.importCount--
				delete(entry.Paths, pathID)
				if len(entry.Paths) == 0 {
					delete(t.table, prefix)
				}
			}
		}
	} else {
		entry, found := t.table[prefix]
		if !found {
			t.importCount++
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
				t.importCount++
			}
		}
		entry.Paths[pathID] = asPath
		if t.importCount > config.GlobalConf.ImportLimit {
			t.sessionCancellation(notification.ImportLimitError)
		}
	}
}
