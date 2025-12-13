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

func copyEvent(src *FlapEvent) (event FlapEvent, triggered bool) {
	if !src.hasTriggered {
		return
	}
	triggered = true

	// Shallow copy of the struct
	event = *src
	// Copy the slice in the struct
	if len(src.RateSecHistory) > 0 {
		event.RateSecHistory = make([]int, len(src.RateSecHistory))
		copy(event.RateSecHistory, src.RateSecHistory)
	}
	return
}

func GetActiveFlapList() (active []FlapEvent, trackedCount int) {
	aFlap := make([]FlapEvent, 0)
	activeMapLock.RLock()
	defer activeMapLock.RUnlock()
	trackedCount = len(activeMap)
	for _, src := range activeMap {
		event, triggered := copyEvent(src)
		if !triggered {
			continue
		}
		aFlap = append(aFlap, event)
	}
	return aFlap, trackedCount
}

func GetActiveFlapPrefix(prefix netip.Prefix) (FlapEvent, bool) {
	activeMapLock.RLock()
	defer activeMapLock.RUnlock()
	src, found := activeMap[prefix]
	if !found {
		return FlapEvent{}, false
	}
	f, triggered := copyEvent(src)
	if !triggered {
		return FlapEvent{}, false
	}
	return f, true
}
