package monitor

import (
	"FlapAlerted/bgp"
	"FlapAlerted/bgp/table"
	"FlapAlerted/config"
	"net/netip"
)

var (
	programVersion string
)

func SetProgramVersion(v string) {
	programVersion = v
}

func StartMonitoring(conf config.UserConfig) {
	config.GlobalConf = conf

	pathChangeChan := make(chan table.PathChange, 1000)

	go notificationHandler(notificationStartChannel, notificationEndChannel)
	go statTracker()
	go recordPathChanges(pathChangeChan)
	bgp.StartBGP(config.GlobalConf.BgpListenAddress, pathChangeChan)
}

func getEvent(k netip.Prefix) (event FlapEvent, found bool) {
	e, ok := activeMap[k]
	if !ok {
		return FlapEvent{}, false
	}

	if !activeMap[k].IsActive {
		return FlapEvent{}, false
	}

	// Shallow copy of the struct
	event = *e
	// Copy the slice in the struct
	if len(activeMap[k].RateSecHistory) > 0 {
		event.RateSecHistory = make([]int, len(activeMap[k].RateSecHistory))
		copy(event.RateSecHistory, activeMap[k].RateSecHistory)
	}
	return event, true
}

func GetActiveFlapList() ([]FlapEvent, int) {
	aFlap := make([]FlapEvent, 0)
	activeMapLock.RLock()
	for k := range activeMap {
		event, ok := getEvent(k)
		if !ok {
			continue
		}
		aFlap = append(aFlap, event)
	}
	trackedCount := len(activeMap)
	activeMapLock.RUnlock()
	return aFlap, trackedCount
}

func GetActiveFlapPrefix(prefix netip.Prefix) (FlapEvent, bool) {
	activeMapLock.RLock()
	defer activeMapLock.RUnlock()
	f, ok := getEvent(prefix)
	if !ok {
		return FlapEvent{}, false
	}
	return f, true
}
