//go:build mod_log
// +build mod_log

package log

import (
	"FlapAlertedPro/monitor"
	"fmt"
	"log"
)

func init() {
	monitor.RegisterModule(&monitor.Module{
		Name:     "mod_log",
		Callback: logFlap,
	})
}

func logFlap(f *monitor.Flap) {
	var IfaceList string
	for i := range f.LastPath.Asn {

		IfaceList = IfaceList + fmt.Sprint(f.LastPath.Asn) + " "
		if i == 10 {
			IfaceList = IfaceList + "and more..."
			break
		}
	}

	log.Println("Prefix:", f.Cidr, ":", IfaceList, "Path change count:", f.PathChangeCountTotal, "Duration (sec):", f.LastSeen-f.FirstSeen)
	summary := monitor.GetActiveFlaps()
	var summaryText string
	for i := range summary {
		summaryText = summaryText + " " + summary[i].Cidr
	}
	log.Println("Summary of currently active flaps:", summaryText)

}
