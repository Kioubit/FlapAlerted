////go:build mod_httpAPI
//// +build mod_httpAPI

package httpAPI

import (
	"FlapAlertedPro/monitor"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
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

func startComplete() {
	http.HandleFunc("/capabilities", showCapabilities)
	//http.Handle("/", http.FileServer(http.FS(dashboardContent)))
	http.Handle("/", dashBoardHandler())
	http.HandleFunc("/flaps/active", getActiveFlaps)
	http.HandleFunc("/flaps/active/compact", activeFlapsCompact)
	http.HandleFunc("/flaps/metrics", metrics)
	http.HandleFunc("/flaps/metrics/prometheus", prometheus)
	err := http.ListenAndServe(":8699", nil)
	if err != nil {
		log.Println("["+moduleName+"] Error starting HTTP api server", err.Error())
	}
}

func dashBoardHandler() http.Handler {
	fSys := fs.FS(dashboardContent)
	html, _ := fs.Sub(fSys, "dashboard")
	return http.FileServer(http.FS(html))
}

func showCapabilities(w http.ResponseWriter, req *http.Request) {
	caps := monitor.GetCapabilities()
	b, err := json.Marshal(caps)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}

func getActiveFlaps(w http.ResponseWriter, req *http.Request) {

	type jsFlap struct {
		Prefix     string
		Paths      []monitor.PathInfo
		FirstSeen  int64
		LastSeen   int64
		TotalCount uint64
	}

	var jsonFlapList = make([]jsFlap, 0)
	activeFlaps := monitor.GetActiveFlaps()
	for i := range activeFlaps {
		pathList := make([]monitor.PathInfo, 0, len(activeFlaps[i].Paths))
		for n := range activeFlaps[i].Paths {
			pathList = append(pathList, *activeFlaps[i].Paths[n])
		}

		jsFlap := jsFlap{
			Prefix:     activeFlaps[i].Cidr,
			FirstSeen:  activeFlaps[i].FirstSeen,
			LastSeen:   activeFlaps[i].LastSeen,
			TotalCount: activeFlaps[i].PathChangeCountTotal,
			Paths:      pathList,
		}
		jsonFlapList = append(jsonFlapList, jsFlap)
	}

	b, err := json.Marshal(jsonFlapList)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}

func activeFlapsCompact(w http.ResponseWriter, req *http.Request) {

	type activeFlapCompact struct {
		Prefix     string
		FirstSeen  int64
		LastSeen   int64
		TotalCount uint64
	}

	var jsonFlapList = make([]activeFlapCompact, 0)
	activeFlaps := monitor.GetActiveFlaps()
	for i := range activeFlaps {
		jsFlap := activeFlapCompact{
			Prefix:     activeFlaps[i].Cidr,
			FirstSeen:  activeFlaps[i].FirstSeen,
			LastSeen:   activeFlaps[i].LastSeen,
			TotalCount: activeFlaps[i].PathChangeCountTotal,
		}
		jsonFlapList = append(jsonFlapList, jsFlap)
	}

	b, err := json.Marshal(jsonFlapList)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}

func metrics(w http.ResponseWriter, req *http.Request) {
	b, err := json.Marshal(monitor.GetMetric())
	if err != nil {
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}

func prometheus(w http.ResponseWriter, req *http.Request) {
	metric := monitor.GetMetric()
	output := fmt.Sprintln("# HELP active_flap_count Number of actively flapping prefixes")
	output += fmt.Sprintln("# TYPE active_flap_count gauge")
	output += fmt.Sprintln("active_flap_count", metric.ActiveFlapCount)

	output += fmt.Sprintln("# HELP active_flap_route_change_count Number of path changes caused by actively flapping prefixes")
	output += fmt.Sprintln("# TYPE active_flap_route_change_count gauge")
	output += fmt.Sprintln("active_flap_route_change_count", metric.ActiveFlapTotalPathChangeCount)

	_, _ = w.Write([]byte(output))
}
