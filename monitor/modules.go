package monitor

import (
	"FlapAlerted/config"
	"log/slog"
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

var notificationStartChannel = make(chan FlapEvent, 10)
var notificationEndChannel = make(chan FlapEvent, 10)

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
	MaxUserDefined       int
}

func GetCapabilities() Capabilities {
	return Capabilities{
		Version: programVersion,
		Modules: getModuleNameList(),
		UserParameters: UserParameters{
			RouteChangeCounter:   config.GlobalConf.RouteChangeCounter,
			OverThresholdTarget:  config.GlobalConf.OverThresholdTarget,
			UnderThresholdTarget: config.GlobalConf.UnderThresholdTarget,
			KeepPathInfo:         config.GlobalConf.KeepPathInfo,
			AddPath:              config.GlobalConf.UseAddPath,
			MaxUserDefined:       config.GlobalConf.MaxUserDefined,
		},
	}
}
