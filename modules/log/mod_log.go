//go:build mod_log
// +build mod_log

package log

import (
	"FlapAlertedPro/monitor"
	"fmt"
	"log"
	"strconv"
)

var moduleName = "mod_log"

func init() {
	monitor.RegisterModule(&monitor.Module{
		Name:     moduleName,
		Callback: logFlap,
	})
}

func logFlap(f *monitor.Flap) {
	var PathList string
	for i := range f.Paths {
		PathList = PathList + fmt.Sprint(f.Paths[i].Asn) + " "
		if i == 10 {
			PathList = PathList + "and " + strconv.Itoa(len(f.Paths)-10) + " more..."
			break
		}
	}

	if len(f.Paths) != 0 {
		log.Println("Prefix:", f.Cidr, " Paths:", PathList, " Path change count:", f.PathChangeCountTotal, "Duration (sec):", f.LastSeen-f.FirstSeen)
	} else {
		log.Println("Prefix:", f.Cidr, " Path change count:", f.PathChangeCountTotal, "Duration (sec):", f.LastSeen-f.FirstSeen)
	}

	summary := monitor.GetActiveFlaps()
	if len(summary) > 1 {
		var summaryText string
		for i := range summary {
			summaryText = summaryText + " " + summary[i].Cidr
		}
		log.Println("Summary of currently active flaps:", summaryText)
	}
	fmt.Println()
}
