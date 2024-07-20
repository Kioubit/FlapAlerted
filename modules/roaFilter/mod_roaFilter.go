//go:build mod_roaFilter

package roaFilter

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"sync"
	"time"
)

var moduleName = "mod_roaFilter"

var source, output *string
var lock sync.Mutex

func init() {
	source = flag.String("roaSource", "", "File path of source ROA JSON")
	output = flag.String("roaOutput", "", "File path of output for the filtered ROA JSON")
	monitor.RegisterModule(&monitor.Module{
		Name:            moduleName,
		CallbackOnce:    change,
		OnStartComplete: started,
	})
}

func started() {
	for {
		// Copy unmodified to output
		filter(*source, *output, "")
		time.Sleep(12 * time.Hour)
	}
}

func change(f *monitor.Flap) {
	// Continue filtering already filtered file
	filter(*output, *output, f.Cidr)
}

func filter(source, output string, cidr string) {
	if source == "" || output == "" {
		slog.Error("roaFilter No source or output files defined")
		return
	}
	if !lock.TryLock() {
		slog.Warn("roaFilter can't keep up", "prefix", cidr)
		return
	}
	defer lock.Unlock()
	inFile, err := os.ReadFile(source)
	if err != nil {
		slog.Error("roaFilter error reading", "file", source)
		return
	}
	if cidr == "" {
		err = os.WriteFile(output, inFile, 0644)
		if err != nil {
			slog.Error("roaFilter error writing", "file", source)
			return
		}
		slog.Info("roaFilter wrote unfiltered ROA file")
		return
	}

	var data map[string]any
	err = json.Unmarshal(inFile, &data)
	if err != nil {
		slog.Error("roaFilter json unmarshal failed", "file", source, "error", err)
		return
	}
	roas, ok := data["roas"]
	if !ok {
		slog.Error("roaFilter json 'roas' key not found", "file", source)
		return
	}
	//fmt.Println(roas)
	prefixList, ok := roas.([]any)
	if !ok {
		slog.Error("roaFilter json 'roas' key is not an array", "file", source)
		return
	}
	tmp := prefixList[:0]
	for _, prefix := range prefixList {
		prefixType, ok := prefix.(map[string]any)
		if !ok {
			slog.Error("roaFilter json 'roas' array elements are not json objects", "file", source)
			return
		}
		prefixKey, ok := prefixType["prefix"]
		if !ok {
			slog.Error("roaFilter json 'prefix' key not found in an element of the 'roas' array", "file", source)
			return
		}
		prefixString, ok := prefixKey.(string)
		if !ok {
			slog.Error("roaFilter json 'prefix' key not a string", "file", source)
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
		slog.Error("roaFilter json marshal failed", "file", source, "error", err)
		return
	}
	err = os.WriteFile(output, outputJson, 0644)
	if err != nil {
		slog.Error("roaFilter error writing", "file", source)
		return
	}
	slog.Info("roaFilter wrote filtered ROA file")
}
