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

var moduleName = "mod_roaFilter"

var roaJsonFile *string
var lock sync.Mutex

func init() {
	roaJsonFile = flag.String("roaJson", "", "File path of source ROA JSON")
	monitor.RegisterModule(&monitor.Module{
		Name:          moduleName,
		CallbackStart: change,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

func change(f monitor.FlapEvent) {
	// Continue filtering already filtered file
	filter(*roaJsonFile, f.Prefix.String())
}

func filter(filePath string, cidr string) {
	logger = logger.With("file", filePath, "prefix", cidr)
	if filePath == "" {
		logger.Error("roaFilter no roaJson file specified")
		return
	}
	if !lock.TryLock() {
		logger.Warn("roaFilter can't keep up")
		return
	}
	defer lock.Unlock()

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
