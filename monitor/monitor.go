package monitor

import (
	"FlapAlerted/analyze"
	"FlapAlerted/bgp"
	"FlapAlerted/config"
	"context"
	"fmt"
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
	userPathChangeChan, notificationChannel := analyze.RecordPathChanges(pathChangeChan)

	wg.Go(func() {
		analyze.RecordUserDefinedMonitors(userPathChangeChan)
	})
	wg.Go(func() {
		statTracker(ctx)
	})
	wg.Go(func() {
		notificationHandler(notificationChannel)
	})
	<-ctx.Done()
	return ctx.Err()
}
