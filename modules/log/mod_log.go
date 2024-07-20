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
		Name:            moduleName,
		CallbackOnce:    logFlapStart,
		CallbackOnceEnd: logFlapEnd,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

func logFlapStart(f *monitor.Flap) {
	logger.Info("event", "type", "start", "prefix", f.Cidr, "first_seen", time.Unix(f.FirstSeen, 0), "total_path_changes", f.PathChangeCountTotal.Load())
}

func logFlapEnd(f *monitor.Flap) {
	logger.Info("event", "type", "end", "prefix", f.Cidr, "first_seen", time.Unix(f.FirstSeen, 0),
		"last_seen", time.Unix(f.LastSeen.Load(), 0), "total_path_changes", f.PathChangeCountTotal.Load())
}
