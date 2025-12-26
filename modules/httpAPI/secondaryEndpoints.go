package httpAPI

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func getActiveFlaps(w http.ResponseWriter, _ *http.Request) {
	activeFlaps := monitor.GetActiveFlapsSummary()

	b, err := json.Marshal(activeFlaps)
	if err != nil {
		logger.Warn("Failed to marshal list to JSON", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(b)
}

func getCapabilities(w http.ResponseWriter, _ *http.Request) {
	b, err := json.Marshal(monitor.GetCapabilities())
	if err != nil {
		logger.Warn("JSON marshal failed for getCapabilities", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(b)
}

func getAvgRouteChanges(w http.ResponseWriter, _ *http.Request) {
	avg := monitor.GetAverageRouteChanges90()
	avgStr := strconv.FormatFloat(avg, 'f', 2, 64)
	_, _ = w.Write([]byte(avgStr))
}

func metrics(w http.ResponseWriter, _ *http.Request) {
	b, err := json.Marshal(monitor.GetMetric())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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

	output += fmt.Sprintln("# HELP route_change_count Number of path changes by all prefixes")
	output += fmt.Sprintln("# TYPE route_change_count gauge")
	output += fmt.Sprintln("route_change_count", metric.TotalPathChangeCount)

	output += fmt.Sprintln("# HELP average_route_changes_90 90th percentile average of route changes over the last 250 seconds, as overall route changes per second")
	output += fmt.Sprintln("# TYPE average_route_changes_90 gauge")
	output += fmt.Sprintln("average_route_changes_90", metric.AverageRouteChanges90)

	output += fmt.Sprintln("# HELP sessions Number of connected BGP feeds")
	output += fmt.Sprintln("# TYPE sessions gauge")
	output += fmt.Sprintln("sessions", metric.Sessions)

	_, _ = w.Write([]byte(output))
}
