//go:build !disable_mod_script

package script

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/exec"
)

var moduleName = "mod_script"
var scriptFileStart *string
var scriptFileEnd *string

func init() {
	scriptFileStart = flag.String("detectionScriptStart", "", "Optional path to script to run when a flap event is detected (start)")
	scriptFileEnd = flag.String("detectionScriptEnd", "", "Optional path to script to run when a flap event is detected (end)")

	monitor.RegisterModule(&monitor.Module{
		Name:            moduleName,
		CallbackStart:   logFlapStart,
		CallbackOnceEnd: logFlapEnd,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

func logFlapStart(f monitor.FlapEvent) {
	runScript(*scriptFileStart, f)
}

func logFlapEnd(f monitor.FlapEvent) {
	runScript(*scriptFileEnd, f)
}

func runScript(path string, f monitor.FlapEvent) {
	if path == "" {
		return
	}
	eventJSON, err := json.Marshal(f)
	if err != nil {
		logger.Error("Marshalling flap information failed", "error", err.Error())
		return
	}
	err = exec.Command(path, string(eventJSON)).Run()
	if err != nil {
		logger.Error("Error executing script", "path", path, "error", err.Error())
	}
}
