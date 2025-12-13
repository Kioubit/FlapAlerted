package monitor

import (
	"FlapAlerted/config"
	"log/slog"
)

type Module struct {
	// Module name
	Name string
	// Function called after the program has started. (Runs in a goroutine)
	OnStartComplete func()
	// Function to register event callbacks that are called in a goroutine
	OnRegisterEventCallbacks func() (callbackStart, callbackEnd func(event FlapEvent))
	eventChan                chan wrappedFlapEvent
}

type wrappedFlapEvent struct {
	event   *FlapEvent
	isStart bool
}

var (
	moduleList     = make([]*Module, 0)
	modulesStarted = false
)

func notificationHandler(c, cEnd <-chan FlapEvent) {
	modulesStarted = true
	moduleCallbackStartComplete()

	for _, m := range moduleList {
		if m.OnRegisterEventCallbacks != nil {
			start, end := m.OnRegisterEventCallbacks()
			m.OnRegisterEventCallbacks = nil
			if start == nil && end == nil {
				continue
			}
			m.eventChan = make(chan wrappedFlapEvent, 200)
			go func(startCb, endCb func(FlapEvent), in <-chan wrappedFlapEvent) {
				for {
					e, ok := <-in
					if !ok {
						return
					}
					if e.isStart && startCb != nil {
						startCb(*e.event)
					} else if !e.isStart && endCb != nil {
						endCb(*e.event)
					}
				}
			}(start, end, m.eventChan)
		}
	}

	defer func() {
		// Cleanup
		for _, m := range moduleList {
			if m.eventChan != nil {
				close(m.eventChan)
			}
		}
	}()

	warningPrinted := false
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
		for _, m := range moduleList {
			if m.eventChan == nil {
				continue
			}
			select {
			case m.eventChan <- wrappedFlapEvent{
				event:   &f,
				isStart: !endNotification,
			}:
			default:
				if !warningPrinted {
					warningPrinted = true
					slog.Warn("one or more modules cannot keep up with event notifications", "first_affected_module", m.Name)
				}
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
