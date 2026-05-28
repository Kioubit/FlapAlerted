package httpAPI

import (
	"FlapAlerted/analyze"
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

func getActivePeers(w http.ResponseWriter, _ *http.Request) {
	activePeers := monitor.GetActivePeersSummary()

	b, err := json.Marshal(activePeers)
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
	w.Header().Set("Content-Type", "text/plain")

	metric := monitor.GetMetric()
	_, _ = fmt.Fprintln(w, "# HELP active_flap_count Number of actively flapping prefixes")
	_, _ = fmt.Fprintln(w, "# TYPE active_flap_count gauge")
	_, _ = fmt.Fprintln(w, "active_flap_count", metric.ActiveFlapCount)

	_, _ = fmt.Fprintln(w, "# HELP active_flap_route_change_count Number of path changes caused by actively flapping prefixes")
	_, _ = fmt.Fprintln(w, "# TYPE active_flap_route_change_count gauge")
	_, _ = fmt.Fprintln(w, "active_flap_route_change_count", metric.ActiveFlapTotalPathChangeCount)

	_, _ = fmt.Fprintln(w, "# HELP route_change_count Number of path changes by all prefixes")
	_, _ = fmt.Fprintln(w, "# TYPE route_change_count gauge")
	_, _ = fmt.Fprintln(w, "route_change_count", metric.TotalPathChangeCount)

	_, _ = fmt.Fprintln(w, "# HELP average_route_changes_90 90th percentile average of route changes over the last 250 seconds, as overall route changes per second")
	_, _ = fmt.Fprintln(w, "# TYPE average_route_changes_90 gauge")
	_, _ = fmt.Fprintln(w, "average_route_changes_90", metric.AverageRouteChanges90)

	_, _ = fmt.Fprintln(w, "# HELP sessions Number of connected BGP feeds")
	_, _ = fmt.Fprintln(w, "# TYPE sessions gauge")
	_, _ = fmt.Fprintln(w, "sessions", metric.Sessions)
}

func prometheusActivePeerRates(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	rates := analyze.GetPeerRates()

	_, _ = fmt.Fprintln(w, "# HELP bgp_updates_per_second BGP update rate per second for each peer")
	_, _ = fmt.Fprintln(w, "# TYPE bgp_updates_per_second gauge")

	_, _ = fmt.Fprintln(w, "# HELP bgp_updates_avg_per_second BGP update rate per second for each peer (averaged)")
	_, _ = fmt.Fprintln(w, "# TYPE bgp_updates_avg_per_second gauge")

	for _, r := range rates {
		if _, err := fmt.Fprintf(
			w, "bgp_updates_per_second{asn=%d} %d\n", r.PeerASN, r.RateSec,
		); err != nil {
			return
		}

		if _, err := fmt.Fprintf(
			w, "bgp_updates_avg_per_second{asn=%d} %.6f\n", r.PeerASN, r.RateSecAvg,
		); err != nil {
			return
		}
	}
}
