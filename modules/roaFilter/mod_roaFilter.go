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
		Name:         moduleName,
		CallbackOnce: change,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

func change(f *monitor.Flap) {
	// Continue filtering already filtered file
	filter(*roaJsonFile, f.Cidr)
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
	inFile, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error("roaFilter error reading file", "error", err)
		return
	}

	var data map[string]any
	err = json.Unmarshal(inFile, &data)
	if err != nil {
		logger.Error("roaFilter json unmarshal failed", "error", err)
		return
	}
	roas, ok := data["roas"]
	if !ok {
		logger.Error("roaFilter json 'roas' key not found")
		return
	}

	prefixList, ok := roas.([]any)
	if !ok {
		logger.Error("roaFilter json 'roas' key is not an array")
		return
	}
	tmp := prefixList[:0]
	for _, prefix := range prefixList {
		prefixType, ok := prefix.(map[string]any)
		if !ok {
			logger.Error("roaFilter json 'roas' array elements are not json objects")
			return
		}
		prefixKey, ok := prefixType["prefix"]
		if !ok {
			logger.Error("roaFilter json 'prefix' key not found in an element of the 'roas' array")
			return
		}
		prefixString, ok := prefixKey.(string)
		if !ok {
			logger.Error("roaFilter json 'prefix' key not a string")
			return
		}

		if prefixString != cidr {
			tmp = append(tmp, prefix)
		}
	}
	prefixList = tmp
	data["roas"] = prefixList
	outputJson, err := json.Marshal(data)
	if err != nil {
		logger.Error("roaFilter json marshal failed", "error", err)
		return
	}
	err = os.WriteFile(filePath, outputJson, 0644)
	if err != nil {
		logger.Error("roaFilter error writing", "error", err)
		return
	}
	logger.Info("roaFilter wrote filtered ROA file")
}
