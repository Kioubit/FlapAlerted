//go:build mod_log

package log

import (
	"FlapAlerted/monitor"
	"log/slog"
	"os"
)

var moduleName = "mod_log"

func init() {
	monitor.RegisterModule(&monitor.Module{
		Name:         moduleName,
		CallbackOnce: logFlap,
	})
}

func logFlap(f *monitor.Flap) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	f.Lock()
	logger.Info("prefix", f.Cidr, "path_change_count", f.PathChangeCountTotal, "first_seen", f.FirstSeen)
	f.Unlock()
}
