package monitor

import (
	"FlapAlerted/config"
	"log/slog"
)

type Module struct {
	// Module name
	Name string
	// Function called after an event starts. (Runs in a goroutine)
	CallbackStart func(event FlapEvent)
	// Function called after an event ends. (Runs in a goroutine)
	CallbackEnd func(event FlapEvent)
	// Function called after the program has started. (Runs in a goroutine)
	OnStartComplete func()
}

var (
	moduleList     = make([]*Module, 0)
	modulesStarted = false
)

func notificationHandler(c, cEnd <-chan FlapEvent) {
	modulesStarted = true
	moduleCallbackStartComplete()

	sem := make(chan struct{}, 20)

	modulesWithCallbacks := make([]*Module, 0)
	for _, module := range moduleList {
		if module.CallbackStart != nil || module.CallbackEnd != nil {
			modulesWithCallbacks = append(modulesWithCallbacks, module)
		}
	}

	for {
		var f FlapEvent
		var ok bool
		endNotification := false
		select {
		case f, ok = <-c:
		case f, ok = <-cEnd:
			endNotification = true
		}
		if !ok {
			return
		}
		for _, m := range modulesWithCallbacks {
			callback := getCallback(m, endNotification)
			if callback == nil {
				continue
			}
			select {
			case sem <- struct{}{}:
				go func() {
					defer func() { <-sem }()
					callback(f)
				}()
			default:
				slog.Warn("module cannot keep up with event notifications", "module", m.Name)
			}
		}
	}
}

func getCallback(m *Module, endNotification bool) func(FlapEvent) {
	if endNotification {
		return m.CallbackEnd
	}
	return m.CallbackStart
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
	RouteChangeCounter       int
	OverThresholdTarget      int
	UnderThresholdTarget     int
	ExpiryRouteChangeCounter int
	KeepPathInfo             bool
	AddPath                  bool
}

func GetCapabilities() Capabilities {
	return Capabilities{
		Version: programVersion,
		Modules: getModuleNameList(),
		UserParameters: UserParameters{
			RouteChangeCounter:       config.GlobalConf.RouteChangeCounter,
			OverThresholdTarget:      config.GlobalConf.OverThresholdTarget,
			UnderThresholdTarget:     config.GlobalConf.UnderThresholdTarget,
			ExpiryRouteChangeCounter: config.GlobalConf.ExpiryRouteChangeCounter,
			KeepPathInfo:             config.GlobalConf.KeepPathInfo,
			AddPath:                  config.GlobalConf.UseAddPath,
		},
	}
}
