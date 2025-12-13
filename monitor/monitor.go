package monitor

import (
	"FlapAlerted/bgp"
	"FlapAlerted/config"
	"context"
	"fmt"
	"net/netip"
	"sync"
)

var (
	programVersion string
)

func SetProgramVersion(v string) {
	programVersion = v
}

func GetProgramVersion() string {
	return programVersion
}

func StartMonitoring(ctx context.Context, conf config.UserConfig) error {
	config.GlobalConf = conf

	var wg sync.WaitGroup
	defer wg.Wait()

	pathChangeChan, err := bgp.StartBGP(ctx, &wg, config.GlobalConf.BgpListenAddress)
	if err != nil {
		return fmt.Errorf("failed to start BGP: %w", err)
	}
	userPathChangeChan, notificationStartChannel, notificationEndChannel := recordPathChanges(pathChangeChan)

	wg.Go(func() {
		recordUserDefinedMonitors(userPathChangeChan)
	})
	wg.Go(func() {
		statTracker(ctx)
	})
	wg.Go(func() {
		notificationHandler(notificationStartChannel, notificationEndChannel)
	})
	<-ctx.Done()
	return ctx.Err()
}

func getEvent(k netip.Prefix) (event FlapEvent, found bool) {
	e, ok := activeMap[k]
	if !ok {
		return FlapEvent{}, false
	}

	if !activeMap[k].hasTriggered {
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
