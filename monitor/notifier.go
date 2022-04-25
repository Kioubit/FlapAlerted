package monitor

import (
	"math"
	"sync"
)

type Module struct {
	Name          string
	Callback      func(*Flap)
	StartComplete func()
}

var (
	moduleList = make([]*Module, 0)
	moduleMu   sync.Mutex
)
var (
	version string
)

func mainNotify(f *Flap) {
	moduleMu.Lock()
	defer moduleMu.Unlock()
	for _, m := range moduleList {
		if m.Callback != nil {
			m.Callback(f)
		}
	}
}

func RegisterModule(module *Module) {
	moduleMu.Lock()
	defer moduleMu.Unlock()
	moduleList = append(moduleList, module)
}

func GetRegisteredModules() []*Module {
	moduleMu.Lock()
	defer moduleMu.Unlock()
	return moduleList
}

func SetVersion(v string) {
	version = v
}

func GetVersion() string {
	return version
}

func GetActiveFlaps() []*Flap {
	aFlap := make([]*Flap, 0)
	flaplistMu.RLock()
	defer flaplistMu.RUnlock()
	for i := range flapList {
		if flapList[i].PathChangeCountTotal >= uint64(NotifyTarget) {
			aFlap = append(aFlap, flapList[i])
		}
	}
	return aFlap
}

type Metric struct {
	ActiveFlapCount                int
	ActiveFlapTotalPathChangeCount uint64
}

func GetMetric() Metric {
	activeFlaps := GetActiveFlaps()

	var totalPathChangeCount uint64
	for i := range activeFlaps {
		totalPathChangeCount = addUint64(totalPathChangeCount, activeFlaps[i].PathChangeCountTotal)
	}

	return Metric{
		ActiveFlapCount:                len(activeFlaps),
		ActiveFlapTotalPathChangeCount: totalPathChangeCount,
	}
}

func GetUserParamterers() (int64, uint64) {
	return FlapPeriod, NotifyTarget
}

func moduleCallback() {
	moduleMu.Lock()
	defer moduleMu.Unlock()
	for _, m := range moduleList {
		if m.StartComplete != nil {
			go m.StartComplete()
		}
	}
}

func addUint64(left, right uint64) uint64 {
	if left > math.MaxUint64-right {
		return math.MaxUint64
	}
	return left + right
}
