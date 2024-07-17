//go:build !no_httpAPI

package httpAPI

import (
	"FlapAlerted/config"
	"FlapAlerted/monitor"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"net/netip"
	"sync"
	"time"
)

var moduleName = "mod_httpAPI"

//go:embed dashboard/*
var dashboardContent embed.FS

func init() {
	monitor.RegisterModule(&monitor.Module{
		Name:          moduleName,
		StartComplete: startComplete,
	})
}

var FlapHistoryMap = make(map[string][]uint64)
var FlapHistoryMapMu sync.RWMutex

func monitorFlap() {
	for {
		<-time.After(10 * time.Second)
		FlapHistoryMapMu.Lock()
		flapList := monitor.GetActiveFlaps()
		for _, f := range flapList {
			f.RLock()
			obj := FlapHistoryMap[f.Cidr]
			if obj == nil {
				FlapHistoryMap[f.Cidr] = []uint64{f.PathChangeCountTotal}
			} else {
				if len(FlapHistoryMap[f.Cidr]) > 500 {
					FlapHistoryMap[f.Cidr] = FlapHistoryMap[f.Cidr][1:]
				}
				FlapHistoryMap[f.Cidr] = append(FlapHistoryMap[f.Cidr], f.PathChangeCountTotal)
			}
			f.RUnlock()
		}
		FlapHistoryMapMu.Unlock()
	}
}

func cleanupHistory() {
	for {
		time.Sleep(1 * time.Duration(config.GlobalConf.FlapPeriod+5) * time.Second)
		FlapHistoryMapMu.Lock()
		newFlapHistoryMap := make(map[string][]uint64)
		flapList := monitor.GetActiveFlaps()
		for _, f := range flapList {
			f.RLock()
			obj := FlapHistoryMap[f.Cidr]
			if obj != nil {
				newFlapHistoryMap[f.Cidr] = obj
			}
			f.RUnlock()
		}
		FlapHistoryMap = newFlapHistoryMap
		FlapHistoryMapMu.Unlock()
	}
}

func startComplete() {
	go monitorFlap()
	go cleanupHistory()
	go streamServe()

	http.Handle("/", dashBoardHandler())
	http.HandleFunc("/capabilities", showCapabilities)
	http.HandleFunc("/flaps/prefix", getPrefix)
	http.HandleFunc("/flaps/active/compact", getActiveFlaps)
	http.HandleFunc("/flaps/statStream", getStatisticStream)
	http.HandleFunc("/flaps/active/history", getFlapHistory)
	http.HandleFunc("/flaps/metrics/json", metrics)
	http.HandleFunc("/flaps/metrics/prometheus", prometheus)
	err := http.ListenAndServe(":8699", nil)
	if err != nil {
		log.Println("["+moduleName+"] Error starting HTTP api server", err.Error())
	}
}

func getFlapHistory(w http.ResponseWriter, req *http.Request) {
	cidr := req.URL.Query().Get("cidr")
	if cidr == "" {
		_, _ = w.Write([]byte("GET request: cidr value missing"))
	}

	FlapHistoryMapMu.RLock()
	result := FlapHistoryMap[cidr]
	if result == nil {
		result = make([]uint64, 0)
	}

	marshaled, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(500)
	} else {
		_, _ = w.Write(marshaled)
	}
	FlapHistoryMapMu.RUnlock()
}

func dashBoardHandler() http.Handler {
	fSys := fs.FS(dashboardContent)
	html, _ := fs.Sub(fSys, "dashboard")
	return http.FileServer(http.FS(html))
}

func showCapabilities(w http.ResponseWriter, _ *http.Request) {
	caps := monitor.GetCapabilities()
	b, err := json.Marshal(caps)
	if err != nil {
		slog.Warn("JSON marshal failed for showCapabilities", "error", err)
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}

func getActiveFlaps(w http.ResponseWriter, _ *http.Request) {
	type jsFlap struct {
		Prefix     string
		FirstSeen  int64
		LastSeen   int64
		TotalCount uint64
	}

	var jsonFlapList = make([]jsFlap, 0)
	activeFlaps := monitor.GetActiveFlaps()
	for _, f := range activeFlaps {
		f.RLock()
		instance := jsFlap{
			Prefix:     f.Cidr,
			FirstSeen:  f.FirstSeen,
			LastSeen:   f.LastSeen,
			TotalCount: f.PathChangeCountTotal,
		}
		f.RUnlock()
		jsonFlapList = append(jsonFlapList, instance)
	}

	b, err := json.Marshal(jsonFlapList)
	if err != nil {
		slog.Warn("Failed to marshal list to JSON", "error", err)
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}

func getPrefix(w http.ResponseWriter, r *http.Request) {
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Invalid prefix"))
		return
	}

	flaps := monitor.GetActiveFlaps()
	for _, f := range flaps {
		f.RLock()
		if f.Cidr == prefix.String() {
			pathList := make([]monitor.PathInfo, 0, len(f.Paths))
			for n := range f.Paths {
				pathList = append(pathList, *f.Paths[n])
			}
			js, err := json.Marshal(struct {
				Prefix     string
				FirstSeen  int64
				LastSeen   int64
				TotalCount uint64
				Paths      []monitor.PathInfo
			}{
				f.Cidr,
				f.FirstSeen,
				f.LastSeen,
				f.PathChangeCountTotal,
				pathList,
			})
			if err != nil {
				w.WriteHeader(500)
				slog.Warn("Failed to marshal prefix to JSON", "error", err)
				_, _ = w.Write([]byte("Internal error"))
			} else {
				_, _ = w.Write(js)
			}
			f.RUnlock()
			return
		}
		f.RUnlock()
	}
	_, _ = w.Write([]byte("null"))
}

func metrics(w http.ResponseWriter, _ *http.Request) {
	b, err := json.Marshal(monitor.GetMetric())
	if err != nil {
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}

func prometheus(w http.ResponseWriter, _ *http.Request) {
	metric := monitor.GetMetric()
	output := fmt.Sprintln("# HELP active_flap_count Number of actively flapping prefixes")
	output += fmt.Sprintln("# TYPE active_flap_count gauge")
	output += fmt.Sprintln("active_flap_count", metric.ActiveFlapCount)

	output += fmt.Sprintln("# HELP active_flap_route_change_count Number of path changes caused by actively flapping prefixes")
	output += fmt.Sprintln("# TYPE active_flap_route_change_count gauge")
	output += fmt.Sprintln("active_flap_route_change_count", metric.ActiveFlapTotalPathChangeCount)

	_, _ = w.Write([]byte(output))
}
