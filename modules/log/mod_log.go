//go:build !disable_mod_log

package log

import (
	"FlapAlerted/monitor"
	"log/slog"
	"os"
	"time"
)

var moduleName = "mod_log"

func init() {
	monitor.RegisterModule(&monitor.Module{
		Name:          moduleName,
		CallbackStart: logFlapStart,
		CallbackEnd:   logFlapEnd,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

func logFlapStart(f monitor.FlapEvent) {
	logger.Info("event", "type", "start", "prefix", f.Prefix.String(), "first_seen", f.FirstSeen, "total_path_changes", f.TotalPathChanges)
}

func logFlapEnd(f monitor.FlapEvent) {
	logger.Info("event", "type", "end", "prefix", f.Prefix.String(), "duration", time.Since(f.FirstSeen).Seconds(), "total_path_changes", f.TotalPathChanges)
}
