//go:build mod_log

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
		Name:         moduleName,
		CallbackOnce: logFlap,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

func logFlap(f *monitor.Flap) {
	logger.Info("event", "prefix", f.Cidr, "first_seen", time.Unix(f.FirstSeen, 0))
}
