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
	eventChan chan []FlapEventNotification
}

func (w *moduleWorker) run() {
	for {
		events, ok := <-w.eventChan
		if !ok {
			return
		}
		for _, e := range events {
			w.impl.OnEvent(e.event, e.isStart)
		}
	}
}

func notificationHandler(c <-chan []FlapEventNotification) {
	modulesStarted.Store(true)

	workerList := make([]*moduleWorker, 0)
	for _, m := range moduleList {
		subscribeToEvents := m.OnStart()
		if subscribeToEvents {
			worker := &moduleWorker{
				impl:      m,
				eventChan: make(chan []FlapEventNotification, 3),
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
		events, ok := <-c
		if !ok {
			return
		}
		for _, w := range workerList {
			select {
			case w.eventChan <- events:
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
	MaxPathHistory           int
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
			MaxPathHistory:           config.GlobalConf.MaxPathHistory,
			AddPath:                  config.GlobalConf.UseAddPath,
		},
	}
}
