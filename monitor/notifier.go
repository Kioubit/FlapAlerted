package monitor

import (
	"FlapAlerted/config"
	"log/slog"
)

type Module struct {
	Name          string
	Callback      func(*Flap)
	CallbackOnce  func(*Flap)
	StartComplete func()
}

var (
	moduleList     = make([]*Module, 0)
	modulesStarted = false
)
var (
	version string
)

func notificationHandler(c chan *Flap) {
	modulesStarted = true
	moduleCallbackStartComplete()
	for {
		f := <-c
		for _, m := range moduleList {
			if m.Callback != nil {
				go m.Callback(f)
			}
			if m.CallbackOnce != nil {
				if !f.meetsMinimumAge.Load() {
					continue
				}
				if !f.notifiedOnce.CompareAndSwap(false, true) {
					continue
				}
				go m.CallbackOnce(f)
			}
		}
	}
}

func RegisterModule(module *Module) {
	if modulesStarted {
		slog.Error("cannot register module", "name", module.Name)
		return
	}
	moduleList = append(moduleList, module)
}

func moduleCallbackStartComplete() {
	for _, m := range moduleList {
		if m.StartComplete != nil {
			go m.StartComplete()
		}
	}
}

func getModuleNameList() []string {
	moduleNameList := make([]string, len(moduleList))
	for i := range moduleList {
		moduleNameList[i] = moduleList[i].Name
	}
	return moduleNameList
}

func GetRegisteredModules() []*Module {
	return moduleList
}

func SetVersion(v string) {
	version = v
}

func GetActiveFlaps() []*Flap {
	return getActiveFlapList()
}

type Metric struct {
	ActiveFlapCount                int
	ActiveFlapTotalPathChangeCount uint64
}

func GetMetric() Metric {
	var activeFlapCount = 0
	var pathChangeCount uint64 = 0
	stats := GetStats()
	if len(stats) != 0 {
		activeFlapCount = stats[len(stats)-1].Active
		pathChangeCount = stats[len(stats)-1].Changes
	}

	return Metric{
		ActiveFlapCount:                activeFlapCount,
		ActiveFlapTotalPathChangeCount: pathChangeCount,
	}
}

type Capabilities struct {
	Version        string
	Modules        []string
	UserParameters UserParameters
}

type UserParameters struct {
	FlapPeriod          int
	RouteChangeCounter  int
	MinimumAge          int
	KeepPathInfo        bool
	AddPath             bool
	RelevantAsnPosition int
}

func GetCapabilities() Capabilities {
	return Capabilities{
		Version: version,
		Modules: getModuleNameList(),
		UserParameters: UserParameters{
			FlapPeriod:          config.GlobalConf.FlapPeriod,
			RouteChangeCounter:  config.GlobalConf.RouteChangeCounter,
			KeepPathInfo:        config.GlobalConf.KeepPathInfo,
			AddPath:             config.GlobalConf.KeepPathInfo,
			RelevantAsnPosition: config.GlobalConf.RelevantAsnPosition,
			MinimumAge:          config.GlobalConf.MinimumAge,
		},
	}
}

func GetStats() []statistic {
	statListLock.RLock()
	defer statListLock.RUnlock()
	result := make([]statistic, len(statList))
	for i := range statList {
		result[i] = statList[i]
	}
	return result
}

func SubscribeToStats() chan statistic {
	return addStatSubscriber()
}
