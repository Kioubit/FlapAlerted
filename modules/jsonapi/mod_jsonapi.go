//go:build mod_jsonapi
// +build mod_jsonapi

package jsonapi

import (
	"FlapAlertedPro/bgp"
	"FlapAlertedPro/monitor"
	"encoding/json"
	"net/http"
)

func init() {
	monitor.RegisterModule(&monitor.Module{
		Name:          "mod_jsonapi",
		StartComplete: startComplete,
	})
}

func startComplete() {
	http.HandleFunc("/flaps/active", activeFlapsJson)
	http.ListenAndServe(":8699", nil)
}

type activeFlap struct {
	Prefix     string
	Paths      []bgp.AsPath
	FirstSeen  int64
	LastSeen   int64
	Count      uint64
	TotalCount uint64
}

func activeFlapsJson(w http.ResponseWriter, req *http.Request) {
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
