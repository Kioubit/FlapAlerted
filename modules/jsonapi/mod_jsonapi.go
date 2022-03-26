//go:build mod_jsonapi
// +build mod_jsonapi

package jsonapi

import (
	"FlapAlertedPro/bgp"
	"FlapAlertedPro/monitor"
	"encoding/json"
	"log"
	"math"
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
	w.Write(b)
}

func metrics(w http.ResponseWriter, req *http.Request) {
	type metric struct {
		ActiveFlapCount            int
		ActiveFlapTotalUpdateCount uint64
	}
	activeFlaps := monitor.GetActiveFlaps()

	var totalUpdateCount uint64
	for i := range activeFlaps {
		totalUpdateCount = addUint64(totalUpdateCount, activeFlaps[i].PathChangeCountTotal)
	}

	newMetric := metric{
		ActiveFlapCount:            len(activeFlaps),
		ActiveFlapTotalUpdateCount: totalUpdateCount,
	}

	b, err := json.Marshal(newMetric)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Write(b)

}

func addUint64(left, right uint64) uint64 {
	if left > math.MaxUint64-right {
		return math.MaxUint64
	}
	return left + right
}
