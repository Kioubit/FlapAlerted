//go:build mod_log
// +build mod_log

package log

import (
	"FlapAlerted/monitor"
	"fmt"
	"log"
)

var moduleName = "mod_log"

func init() {
	monitor.RegisterModule(&monitor.Module{
		Name:         moduleName,
		CallbackOnce: logFlap,
	})
}

func logFlap(f *monitor.Flap) {
	log.Println("Prefix:", f.Cidr, " Path change count:", f.PathChangeCountTotal, "Last seen:", f.LastSeen)

	summary := monitor.GetActiveFlaps()
	if len(summary) > 1 {
		var summaryText string
		for i := range summary {
			summary[i].RLock()
			summaryText = summaryText + " " + summary[i].Cidr
			summary[i].RUnlock()
		}
		log.Println("Summary of currently active flaps:", summaryText)
	}
	fmt.Println()
}
