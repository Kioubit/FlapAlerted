//go:build mod_roaFilter

package roaFilter

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"sync"
)

var (
	roaJsonFile = flag.String("roaJson", "", "File path of source ROA JSON")
)

type Module struct {
	name   string
	logger *slog.Logger
	lock   sync.Mutex
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) OnStart() bool {
	if *roaJsonFile == "" {
		return false
	}
	m.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", m.Name())
	return true
}

func (m *Module) OnEvent(f monitor.FlapEvent, isStart bool) {
	if isStart {
		m.filter(*roaJsonFile, f.Prefix.String())
	}
}

func init() {
	monitor.RegisterModule(&Module{
		name: "mod_roaFilter",
	})
}

func (m *Module) filter(filePath string, cidr string) {
	logger := m.logger.With("file", filePath, "prefix", cidr)
	if filePath == "" {
		logger.Error("no roaJson file specified")
		return
	}
	if !m.lock.TryLock() {
		logger.Warn("roaFilter can't keep up")
		return
	}
	defer m.lock.Unlock()

	data, err := readROAFile(filePath)
	if err != nil {
		logger.Error("failed to read ROA file", "error", err)
		return
	}

	roas, ok := data["roas"].([]any)
	if !ok {
		logger.Error("invalid 'roas' field")
		return
	}

	filtered := filterPrefix(roas, cidr)
	data["roas"] = filtered

	if err := writeROAFile(filePath, data); err != nil {
		logger.Error("failed to write ROA file", "error", err)
		return
	}

	logger.Info("wrote filtered ROA file")
}

func readROAFile(path string) (map[string]any, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	err = json.Unmarshal(file, &data)
	return data, err
}

func filterPrefix(roas []any, cidr string) []any {
	filtered := make([]any, 0, len(roas))
	for _, roa := range roas {
		prefixObj, ok := roa.(map[string]any)
		if !ok || prefixObj["prefix"] != cidr {
			filtered = append(filtered, roa)
		}
	}
	return filtered
}

func writeROAFile(path string, data map[string]any) error {
	output, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, output, 0644)
}
