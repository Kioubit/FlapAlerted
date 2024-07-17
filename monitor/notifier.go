package monitor

import (
	"FlapAlerted/config"
	"sync"
)

type Module struct {
	Name          string
	Callback      func(*Flap)
	CallbackOnce  func(*Flap)
	StartComplete func()
}

var (
	moduleList = make([]*Module, 0)
	moduleMu   sync.Mutex
)
var (
	version string
)

func notificationHandler(c chan *Flap) {
	for {
		f := <-c
		moduleMu.Lock()
		for _, m := range moduleList {
			if m.Callback != nil {
				m.Callback(f)
			}
			if m.CallbackOnce != nil {
				if f.PathChangeCountTotal > uint64(config.GlobalConf.RouteChangeCounter) {
					return
				}
				m.CallbackOnce(f)
			}
		}
		moduleMu.Unlock()
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
	FlapPeriod          int64
	NotifyTarget        int
	KeepPathInfo        bool
	AddPath             bool
	RelevantAsnPosition int
	NotifyOnce          bool
}

func GetCapabilities() Capabilities {
	uParams := UserParameters{
		FlapPeriod:          config.GlobalConf.FlapPeriod,
		NotifyTarget:        config.GlobalConf.RouteChangeCounter,
		KeepPathInfo:        config.GlobalConf.KeepPathInfo,
		AddPath:             config.GlobalConf.KeepPathInfo,
		RelevantAsnPosition: config.GlobalConf.RelevantAsnPosition,
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
