package monitor

import (
	"FlapAlerted/bgp"
	"FlapAlerted/bgp/table"
	"FlapAlerted/config"
	"net/netip"
)

func StartMonitoring(conf config.UserConfig) {
	config.GlobalConf = conf

	pathChangeChan := make(chan table.PathChange, 1000)

	go notificationHandler(NotificationChannel, NotificationEndChannel)
	go statTracker()
	go recordPathChanges(pathChangeChan)
	bgp.StartBGP(config.GlobalConf.BgpListenAddress, pathChangeChan)
}

func getEvent(k netip.Prefix) (event FlapEvent, found bool) {
	e, ok := activeMap[k] // shallow copy of the struct
	if !ok {
		return FlapEvent{}, false
	}

	if !activeMap[k].IsActive {
		return FlapEvent{}, false
	}

	if activeMap[k].overThresholdCount < config.GlobalConf.OverThresholdTarget {
		return FlapEvent{}, false
	}

	event = *e
	if len(activeMap[k].RateSecHistory) > 0 {
		event.RateSecHistory = make([]int, len(activeMap[k].RateSecHistory))
		copy(event.RateSecHistory, activeMap[k].RateSecHistory)
	}
	return event, true
}

func GetActiveFlapList() ([]FlapEvent, int) {
	aFlap := make([]FlapEvent, 0)
	activeMapLock.RLock()
	belowThreshold := 0
	for k := range activeMap {
		event, ok := getEvent(k)
		if !ok {
			belowThreshold++
			continue
		}
		aFlap = append(aFlap, event)
	}
	activeMapLock.RUnlock()
	return aFlap, belowThreshold
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
