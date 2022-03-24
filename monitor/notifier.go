package monitor

import "sync"

type Module struct {
	Name          string
	Callback      func(*Flap)
	StartComplete func()
}

var (
	moduleList = make([]*Module, 0)
	moduleMu   sync.Mutex
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

func GetActiveFlaps() []*Flap {
	aFlap := make([]*Flap, 0)
	flaplistMu.RLock()
	defer flaplistMu.RUnlock()
	for i := range flapList {
		if flapList[i].PathChangeCountTotal >= uint64(NotifyTarget) {
			aFlap = append(aFlap, flapList[i])
		}
	}
	return aFlap
}

func GetUserParamterers() (int64, uint64) {
	return FlapPeriod, NotifyTarget
}
