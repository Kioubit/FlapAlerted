//go:build !disable_mod_script

package script

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"os/exec"
)

var (
	scriptFileStart = flag.String("detectionScriptStart", "", "Optional path to script to run when a flap event is detected (start)")
	scriptFileEnd   = flag.String("detectionScriptEnd", "", "Optional path to script to run when a flap event is detected (end)")
)

type Module struct {
	name   string
	logger *slog.Logger
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) OnStart() bool {
	if *scriptFileStart == "" && *scriptFileEnd == "" {
		return false
	}

	m.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", m.Name())
	return true
}

func (m *Module) OnEvent(f analyze.FlapEvent, isStart bool) {
	if isStart {
		m.runScript(*scriptFileStart, f)
	} else {
		m.runScript(*scriptFileEnd, f)
	}
}

func (m *Module) runScript(path string, f analyze.FlapEvent) {
	if path == "" {
		return
	}
	l := m.logger.With("path", path, "prefix", f.Prefix)
	eventJSON, err := json.Marshal(analyze.FlapEventNoPaths(f))
	if err != nil {
		l.Error("Marshalling flap information failed", "error", err.Error())
		return
	}
	err = exec.Command(path, string(eventJSON)).Run()
	if err != nil {
		l.Error("Error executing script", "error", err.Error())
	}
}

func init() {
	monitor.RegisterModule(&Module{
		name: "mod_script",
	})
}
