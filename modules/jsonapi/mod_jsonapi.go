//go:build mod_jsonapi
// +build mod_jsonapi

package jsonapi

import (
	"FlapAlertedPro/bgp"
	"FlapAlertedPro/monitor"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

var moduleName = "mod_jsonapi"

func init() {
	monitor.RegisterModule(&monitor.Module{
		Name:          moduleName,
		StartComplete: startComplete,
	})
}

func startComplete() {
	http.HandleFunc("/flaps/active", activeFlaps)
	http.HandleFunc("/flaps/metrics", metrics)
	http.HandleFunc("/flaps/metrics/prometheus", prometheus)
	err := http.ListenAndServe(":8699", nil)
	if err != nil {
		log.Println("["+moduleName+"] Error starting JSON api server", err.Error())
	}
}

type activeFlap struct {
	Prefix     string
	Paths      []bgp.AsPath
	FirstSeen  int64
	LastSeen   int64
	Count      uint64
	TotalCount uint64
}

func activeFlaps(w http.ResponseWriter, req *http.Request) {
	var jsonFlapList = make([]activeFlap, 0)
	activeFlaps := monitor.GetActiveFlaps()
	for i := range activeFlaps {
		jsFlap := activeFlap{
			Prefix:     activeFlaps[i].Cidr,
			FirstSeen:  activeFlaps[i].FirstSeen,
			LastSeen:   activeFlaps[i].LastSeen,
			TotalCount: activeFlaps[i].PathChangeCountTotal,
			Paths:      activeFlaps[i].Paths,
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
	var output string
	output = fmt.Sprintln("# HELP active_flap_count Number of actively flapping prefixes")
	output = output + fmt.Sprintln("# TYPE active_flap_count gauge")
	output = output + fmt.Sprintln("active_flap_count", metric.ActiveFlapCount)

	output = fmt.Sprintln("# HELP active_flap_update_count Number route updates caused by actively flapping prefixes")
	output = output + fmt.Sprintln("# TYPE active_flap_update_count gauge")
	output = output + fmt.Sprintln("active_flap_update_count", metric.ActiveFlapTotalUpdateCount)

	_, _ = w.Write([]byte(output))
}
