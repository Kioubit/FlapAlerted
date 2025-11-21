package monitor

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/config"
	"log/slog"
	"strconv"
)

type Module struct {
	Name            string
	CallbackStart   func(event FlapEvent)
	CallbackOnceEnd func(event FlapEvent)
	OnStartComplete func()
}

var (
	moduleList     = make([]*Module, 0)
	modulesStarted = false
)
var (
	version string
)

func notificationHandler(c, cEnd chan FlapEvent) {
	modulesStarted = true
	moduleCallbackStartComplete()
	for {
		var f FlapEvent
		endNotification := false
		select {
		case f = <-c:
		case f = <-cEnd:
			endNotification = true
		}
		for _, m := range moduleList {
			if endNotification {
				if m.CallbackOnceEnd != nil {
					go m.CallbackOnceEnd(f)
				}
				continue
			}
			if m.CallbackStart != nil {
				go m.CallbackStart(f)
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
		if m.OnStartComplete != nil {
			go m.OnStartComplete()
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

func GetRegisteredModuleNames() []string {
	list := make([]string, 0)
	for _, m := range moduleList {
		list = append(list, m.Name)
	}
	return list
}

func SetVersion(v string) {
	version = v
}

func GetActiveFlaps() []FlapEvent {
	active, _ := GetActiveFlapList()
	return active
}

func GetActiveFlapsSummary() []FlapSummary {
	l := lastFlapSummaryList.Load()
	if l == nil {
		return make([]FlapSummary, 0)
	}
	return *l
}

type Metric struct {
	ActiveFlapCount                int
	ActiveFlapTotalPathChangeCount uint64
	AverageRouteChanges90          string
	Sessions                       int
}

func GetMetric() Metric {
	var activeFlapCount = 0
	var pathChangeCount uint64 = 0
	stats := GetStats()
	if len(stats) != 0 {
		activeFlapCount = stats[len(stats)-1].Stats.Active
		pathChangeCount = stats[len(stats)-1].Stats.Changes
	}
	avg := GetAverageRouteChanges90()
	avgStr := strconv.FormatFloat(avg, 'f', 2, 64)

	return Metric{
		ActiveFlapCount:                activeFlapCount,
		ActiveFlapTotalPathChangeCount: pathChangeCount,
		AverageRouteChanges90:          avgStr,
		Sessions:                       common.GetSessionCount(),
	}
}

type Capabilities struct {
	Version        string
	Modules        []string
	UserParameters UserParameters
}

type UserParameters struct {
	RouteChangeCounter   int
	OverThresholdTarget  int
	UnderThresholdTarget int
	KeepPathInfo         bool
	AddPath              bool
}

func GetCapabilities() Capabilities {
	return Capabilities{
		Version: version,
		Modules: getModuleNameList(),
		UserParameters: UserParameters{
			RouteChangeCounter:   config.GlobalConf.RouteChangeCounter,
			OverThresholdTarget:  config.GlobalConf.OverThresholdTarget,
			UnderThresholdTarget: config.GlobalConf.UnderThresholdTarget,
			KeepPathInfo:         config.GlobalConf.KeepPathInfo,
			AddPath:              config.GlobalConf.UseAddPath,
		},
	}
}
