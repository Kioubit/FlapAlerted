//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/config"
	"FlapAlerted/monitor"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"
)

var moduleName = "mod_httpAPI"

//go:embed dashboard/*
var dashboardContent embed.FS

var limitedHttpAPI *bool
var httpAPIListenAddress *string
var gageMaxValue *int

func init() {
	limitedHttpAPI = flag.Bool("limitedHttpApi", false, "Disable http API endpoints not needed for"+
		" the user interface")
	httpAPIListenAddress = flag.String("httpAPIListenAddress", ":8699", "Listen address for the http api")
	gageMaxValue = flag.Int("httpGageMaxValue", 200, "HTTP dashboard Gage max value")

	monitor.RegisterModule(&monitor.Module{
		Name:            moduleName,
		OnStartComplete: startComplete,
	})
}

var FlapHistoryMap = make(map[string][]uint64)
var FlapHistoryMapMu sync.RWMutex

func monitorFlap() {
	for {
		<-time.After(10 * time.Second)
		FlapHistoryMapMu.Lock()
		flapList := monitor.GetActiveFlapsSummary()
		for _, f := range flapList {
			obj := FlapHistoryMap[f.Prefix]
			if obj == nil {
				FlapHistoryMap[f.Prefix] = []uint64{f.TotalCount}
			} else {
				if len(FlapHistoryMap[f.Prefix]) > 100 {
					FlapHistoryMap[f.Prefix] = FlapHistoryMap[f.Prefix][1:]
				}
				FlapHistoryMap[f.Prefix] = append(FlapHistoryMap[f.Prefix], f.TotalCount)
			}
		}
		FlapHistoryMapMu.Unlock()
	}
}

func cleanupHistory() {
	for {
		time.Sleep(1 * time.Duration(config.GlobalConf.FlapPeriod+5) * time.Second)
		FlapHistoryMapMu.Lock()
		newFlapHistoryMap := make(map[string][]uint64)
		flapList := monitor.GetActiveFlapsSummary()
		for _, f := range flapList {
			obj := FlapHistoryMap[f.Prefix]
			if obj != nil {
				newFlapHistoryMap[f.Prefix] = obj
			}
		}
		FlapHistoryMap = newFlapHistoryMap
		FlapHistoryMapMu.Unlock()
	}
}

func startComplete() {
	go monitorFlap()
	go cleanupHistory()
	go streamServe()

	mux := http.NewServeMux()
	mux.Handle("/", mainPageHandler())
	mux.HandleFunc("/capabilities", showCapabilities)
	mux.HandleFunc("/flaps/prefix", getPrefix)
	mux.HandleFunc("/flaps/statStream", getStatisticStream)
	mux.HandleFunc("/flaps/active/history", getFlapHistory)

	if !*limitedHttpAPI {
		mux.HandleFunc("/flaps/avgRouteChanges90", getAvgRouteChanges)
		mux.HandleFunc("/flaps/active/compact", getActiveFlaps)
		mux.HandleFunc("/flaps/metrics/json", metrics)
		mux.HandleFunc("/flaps/metrics/prometheus", prometheus)
	}

	s := &http.Server{
		Addr:              *httpAPIListenAddress,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
	}
	err := s.ListenAndServe()
	if err != nil {
		slog.Error("["+moduleName+"] Error starting HTTP api server", "error", err)
	}
}

func getAvgRouteChanges(w http.ResponseWriter, _ *http.Request) {
	avg := monitor.GetAverageRouteChanges90()
	avgStr := strconv.FormatFloat(avg, 'f', 2, 64)
	_, _ = w.Write([]byte(avgStr))
}

func getFlapHistory(w http.ResponseWriter, req *http.Request) {
	cidr := req.URL.Query().Get("cidr")
	if cidr == "" {
		_, _ = w.Write([]byte("GET request: cidr value missing"))
	}

	FlapHistoryMapMu.RLock()
	defer FlapHistoryMapMu.RUnlock()
	result := FlapHistoryMap[cidr]
	if result == nil {
		_, _ = w.Write([]byte("null"))
		return
	}

	marshaled, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(500)
	} else {
		_, _ = w.Write(marshaled)
	}
}

func mainPageHandler() http.Handler {
	fSys := fs.FS(dashboardContent)
	html, _ := fs.Sub(fSys, "dashboard")
	return http.FileServer(http.FS(html))
}

func showCapabilities(w http.ResponseWriter, _ *http.Request) {
	caps := monitor.GetCapabilities()

	fullCaps := struct {
		monitor.Capabilities
		GageMaxValue int `json:"gageMaxValue"`
	}{
		Capabilities: caps,
		GageMaxValue: *gageMaxValue,
	}

	b, err := json.Marshal(fullCaps)
	if err != nil {
		slog.Warn("JSON marshal failed for showCapabilities", "error", err)
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}

func getActiveFlaps(w http.ResponseWriter, _ *http.Request) {
	activeFlaps := monitor.GetActiveFlapsSummary()

	b, err := json.Marshal(activeFlaps)
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
		_, _ = w.Write([]byte("null"))
		return
	}

	flaps := monitor.GetActiveFlaps()
	for _, f := range flaps {
		if f.Cidr == prefix.String() {
			f.RLock() // Needed to get paths
			var pathList []monitor.PathInfo
			if config.GlobalConf.KeepPathInfo {
				pathList = make([]monitor.PathInfo, 0, f.Paths.Length())
				for _, v := range f.Paths.All() {
					pathList = append(pathList, *v)
				}
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
				f.LastSeen.Load(),
				f.PathChangeCountTotal.Load(),
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

	output += fmt.Sprintln("# HELP average_route_changes_90 90th percentile average of route changes over the last 250 seconds, as overall route changes per second")
	output += fmt.Sprintln("# TYPE average_route_changes_90 gauge")
	output += fmt.Sprintln("average_route_changes_90", metric.AverageRouteChanges90)

	output += fmt.Sprintln("# HELP sessions Number of connected BGP feeds")
	output += fmt.Sprintln("# TYPE sessions gauge")
	output += fmt.Sprintln("sessions", metric.Sessions)

	_, _ = w.Write([]byte(output))
}
