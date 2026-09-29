//go:build !disable_mod_log

package log

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	"context"
	"flag"
	"log/slog"
	"sync"
)

var (
	disableLog = flag.Bool("logDisable", false, "Disable flap event logging")
)

type Module struct {
	name   string
	logger *slog.Logger
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) OnStart(_ context.Context, _ *sync.WaitGroup, logger *slog.Logger) bool {
	if *disableLog {
		return false
	}
	m.logger = logger
	return true
}

func (m *Module) OnEvent(f analyze.FlapEvent, isStart bool) {
	eventType := "end"
	if isStart {
		eventType = "start"
	}
	m.logger.Info("event", "type", eventType, "prefix", f.Prefix.String(), "first_seen", f.FirstSeen, "total_path_changes", f.TotalPathChanges)
}

func init() {
	monitor.RegisterModule(&Module{
		name: "mod_log",
	})
}
