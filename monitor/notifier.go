package monitor

import (
	"FlapAlerted/config"
	"log/slog"
	"math"
	"sort"
	"strconv"
)

type Module struct {
	Name            string
	Callback        func(*Flap)
	CallbackOnce    func(*Flap)
	CallbackOnceEnd func(*Flap)
	OnStartComplete func()
}

var (
	moduleList     = make([]*Module, 0)
	modulesStarted = false
)
var (
	version string
)

func notificationHandler(c, cEnd chan *Flap) {
	modulesStarted = true
	moduleCallbackStartComplete()
	for {
		var f *Flap
		endNotification := false
		select {
		case f = <-c:
		case f = <-cEnd:
			endNotification = true
		}
		for _, m := range moduleList {
			if endNotification {
				if !f.notifiedOnce.Load() {
					continue
				}
				if m.CallbackOnceEnd != nil {
					go m.CallbackOnceEnd(f)
				}
				continue
			}
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

func GetActiveFlaps() []*Flap {
	return getActiveFlapList()
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
	}
}

type Capabilities struct {
	Version        string
	Modules        []string
	UserParameters UserParameters
}

type UserParameters struct {
	FlapPeriod               int
	RouteChangeCounter       int
	MinimumAge               int
	KeepPathInfo             bool
	KeepPathInfoDetectedOnly bool
	AddPath                  bool
	RelevantAsnPosition      int
}

func GetCapabilities() Capabilities {
	return Capabilities{
		Version: version,
		Modules: getModuleNameList(),
		UserParameters: UserParameters{
			FlapPeriod:               config.GlobalConf.FlapPeriod,
			RouteChangeCounter:       config.GlobalConf.RouteChangeCounter,
			KeepPathInfo:             config.GlobalConf.KeepPathInfo,
			KeepPathInfoDetectedOnly: config.GlobalConf.KeepPathInfoDetectedOnly,
			AddPath:                  config.GlobalConf.UseAddPath,
			RelevantAsnPosition:      config.GlobalConf.RelevantAsnPosition,
			MinimumAge:               config.GlobalConf.MinimumAge,
		},
	}
}

func GetStats() []statisticWrapper {
	statListLock.RLock()
	defer statListLock.RUnlock()
	result := make([]statisticWrapper, len(statList))
	for i := range statList {
		result[i] = statisticWrapper{
			List:     nil,
			Stats:    statList[i],
			Sessions: -1,
		}
	}
	if len(statList) > 0 {
		l := lastFlapSummaryList.Load()
		if l != nil {
			result[len(statList)-1].List = *l
		}
	}
	return result
}

func SubscribeToStats() chan statisticWrapper {
	return addStatSubscriber()
}

func GetAverageRouteChanges90() float64 {
	stats := GetStats()
	changesList := make([]uint64, len(stats))
	for i, stat := range stats {
		changesList[i] = stat.Stats.Changes
	}
	sort.Slice(changesList, func(i, j int) bool { return changesList[i] < changesList[j] })
	cutLength := int(math.Ceil(float64(len(changesList)) * 0.90))
	changesList = changesList[:cutLength]

	if len(changesList) == 0 {
		return 0
	}

	var sum uint64 = 0
	for _, u := range changesList {
		sum = addUint64(sum, u)
	}
	avg := float64(sum) / float64(len(changesList))
	return avg / 5 // Data collected for 5 seconds
}

func addUint64(left, right uint64) uint64 {
	if left > math.MaxUint64-right {
		return math.MaxUint64
	}
	return left + right
}
