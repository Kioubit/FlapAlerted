package monitor

import (
	"FlapAlerted/config"
	"log/slog"
	"sync/atomic"
)

var (
	moduleList     = make([]Module, 0)
	modulesStarted atomic.Bool
)

type Module interface {
	// Name Module name
	Name() string

	// OnStart is called before the monitoring starts
	// Implementation should check if it needs to receive events.
	// True must be returned to subscribe to events.
	// Background goroutines may be spawned here if needed as well.
	OnStart() bool

	// OnEvent is called when a flap event occurs.
	// Runs inside a worker goroutine.
	// This is only called if OnStart() returned true.
	OnEvent(event FlapEvent, isStart bool)
}

type moduleWorker struct {
	impl      Module
	eventChan chan wrappedFlapEvent
}

func (w *moduleWorker) run() {
	for {
		e, ok := <-w.eventChan
		if !ok {
			return
		}
		w.impl.OnEvent(e.event, e.isStart)
	}
}

type wrappedFlapEvent struct {
	event   FlapEvent
	isStart bool
}

func notificationHandler(c, cEnd <-chan FlapEvent) {
	modulesStarted.Store(true)

	workerList := make([]*moduleWorker, 0)
	for _, m := range moduleList {
		subscribeToEvents := m.OnStart()
		if subscribeToEvents {
			worker := &moduleWorker{
				impl:      m,
				eventChan: make(chan wrappedFlapEvent, 200),
			}
			go worker.run()
			workerList = append(workerList, worker)
		}
	}

	defer func() {
		// Cleanup
		for _, w := range workerList {
			close(w.eventChan)
		}
	}()

	warningPrinted := false
	for {
		var f FlapEvent
		var ok bool
		isStart := false
		select {
		case f, ok = <-c:
			isStart = true
		case f, ok = <-cEnd:
		}
		if !ok {
			return
		}
		for _, w := range workerList {
			select {
			case w.eventChan <- wrappedFlapEvent{
				event:   f,
				isStart: isStart,
			}:
			default:
				if !warningPrinted {
					warningPrinted = true
					slog.Warn("one or more modules cannot keep up with event notifications", "first_affected_module", w.impl.Name())
				}
			}
		}
	}
}

func RegisterModule(module Module) {
	if modulesStarted.Load() {
		slog.Error("cannot register module", "name", module.Name())
		return
	}
	moduleList = append(moduleList, module)
}

func GetRegisteredModuleNames() []string {
	moduleNameList := make([]string, len(moduleList))
	for i := range moduleList {
		moduleNameList[i] = moduleList[i].Name()
	}
	return moduleNameList
}

// ---------------------------------------------------------------------------------------------------------------------

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
		Modules: GetRegisteredModuleNames(),
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
