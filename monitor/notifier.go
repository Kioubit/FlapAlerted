package monitor

import (
	"FlapAlertedPro/config"
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

func GetActiveFlaps() []Flap {
	return getActiveFlapList()
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

type Capabilities struct {
	Version        string
	Modules        []string
	UserParameters UserParameters
}

type UserParameters struct {
	FlapPeriod          int64
	NotifyTarget        int64
	KeepPathInfo        bool
	AddPath             bool
	RelevantAsnPosition int64
	NotifyOnce          bool
}

func GetCapabilities() Capabilities {
	uParams := UserParameters{
		FlapPeriod:          config.GlobalConf.FlapPeriod,
		NotifyTarget:        config.GlobalConf.RouteChangeCounter,
		KeepPathInfo:        config.GlobalConf.KeepPathInfo,
		AddPath:             config.GlobalConf.KeepPathInfo,
		RelevantAsnPosition: config.GlobalConf.RelevantAsnPosition,
		NotifyOnce:          config.GlobalConf.NotifyOnce,
	}
	return Capabilities{
		Version:        version,
		Modules:        getModuleList(),
		UserParameters: uParams,
	}
}

func getModuleList() []string {
	moduleMu.Lock()
	defer moduleMu.Unlock()
	moduleNameList := make([]string, len(moduleList))
	for i := range moduleList {
		moduleNameList[i] = moduleList[i].Name
	}
	return moduleNameList
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
