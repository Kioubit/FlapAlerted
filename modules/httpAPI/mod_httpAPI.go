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
	"net"
	"net/http"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

var moduleName = "mod_httpAPI"

//go:embed dashboard/*
var dashboardContent embed.FS

var limitedHttpAPI *bool
var httpAPIListenAddress *string
var gageMaxValue *uint
var maxUserDefinedMonitors *uint

func init() {
	limitedHttpAPI = flag.Bool("limitedHttpApi", false, "Disable http API endpoints not needed for"+
		" the user interface & be less friendly to scraping")
	httpAPIListenAddress = flag.String("httpAPIListenAddress", ":8699", "Listen address for the HTTP API (TCP address like :8699 or Unix socket path)")
	gageMaxValue = flag.Uint("httpGageMaxValue", 400, "HTTP dashboard Gage max value")
	maxUserDefinedMonitors = flag.Uint("maxUserDefined", 5, "Maximum number of user-defined tracked prefixes. Use zero to disable")

	monitor.RegisterModule(&monitor.Module{
		Name:            moduleName,
		OnStartComplete: startComplete,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

func startComplete() {
	go streamServe()

	mux := http.NewServeMux()
	mux.Handle("/", mainPageHandler())
	mux.HandleFunc("/capabilities", antiScrapeMiddleware(showCapabilities))
	mux.HandleFunc("/flaps/prefix", antiScrapeMiddleware(getPrefix))
	mux.HandleFunc("/flaps/statStream", getStatisticStream)
	mux.HandleFunc("/flaps/active/history", antiScrapeMiddleware(getFlapHistory))
	mux.HandleFunc("/sessions", antiScrapeMiddleware(getBgpSessions))

	if !*limitedHttpAPI {
		mux.HandleFunc("/flaps/avgRouteChanges90", getAvgRouteChanges)
		mux.HandleFunc("/flaps/active/compact", getActiveFlaps)
		mux.HandleFunc("/flaps/active/roa", getActiveFlapsRoa)
		mux.HandleFunc("/flaps/metrics/json", metrics)
		mux.HandleFunc("/flaps/metrics/prometheus", prometheus)
	}

	if *maxUserDefinedMonitors != 0 {
		mux.HandleFunc("/userDefined/subscribe", getUserDefinedStatisticStream)
		mux.HandleFunc("/userDefined/prefix", antiScrapeMiddleware(getUserDefinedStatistic))
	}

	s := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
	}
	var listener net.Listener
	var err error
	if strings.HasPrefix(*httpAPIListenAddress, "/") {
		_ = os.Remove(*httpAPIListenAddress)
		listener, err = net.Listen("unix", *httpAPIListenAddress)
		if err != nil {
			logger.Error("Error creating Unix listener", "error", err)
			return
		}
	} else {
		listener, err = net.Listen("tcp", *httpAPIListenAddress)
		if err != nil {
			logger.Error("Error creating TCP listener", "error", err)
			return
		}
	}
	err = s.Serve(listener)
	if err != nil {
		logger.Error("Error starting HTTP api server", "error", err)
	}
}

func antiScrapeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !*limitedHttpAPI {
			next(w, r)
			return
		}

		headerValue := r.Header.Get("X-AS")
		timestamp, err := strconv.ParseInt(headerValue, 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		now := time.Now().Unix()
		diff := now - timestamp

		if diff < 0 {
			diff = -diff
		}

		// Check if timestamp is within 1 hour (3600 seconds)
		if diff > 3600 {
			http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
			return
		}

		next(w, r)
	}
}

func getAvgRouteChanges(w http.ResponseWriter, _ *http.Request) {
	avg := monitor.GetAverageRouteChanges90()
	avgStr := strconv.FormatFloat(avg, 'f', 2, 64)
	_, _ = w.Write([]byte(avgStr))
}

func getFlapHistory(w http.ResponseWriter, r *http.Request) {
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("cidr"))
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("null"))
		return
	}
	result, found := monitor.GetActiveFlapPrefix(prefix)
	if !found {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("null"))
		return
	}

	marshaled, err := json.Marshal(result.RateSecHistory)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(marshaled)
}

func mainPageHandler() http.Handler {
	fSys := fs.FS(dashboardContent)
	html, _ := fs.Sub(fSys, "dashboard")
	return http.FileServer(http.FS(html))
}

func showCapabilities(w http.ResponseWriter, _ *http.Request) {
	caps := monitor.GetCapabilities()

	type ModHttpCaps struct {
		GageMaxValue   uint `json:"gageMaxValue"`
		MaxUserDefined uint `json:"maxUserDefined"`
	}

	fullCaps := struct {
		monitor.Capabilities
		ModHttpCaps ModHttpCaps `json:"modHttp"`
	}{
		Capabilities: caps,
		ModHttpCaps: ModHttpCaps{
			GageMaxValue:   *gageMaxValue,
			MaxUserDefined: *maxUserDefinedMonitors,
		},
	}

	b, err := json.Marshal(fullCaps)
	if err != nil {
		logger.Warn("JSON marshal failed for showCapabilities", "error", err)
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}

func getActiveFlaps(w http.ResponseWriter, _ *http.Request) {
	activeFlaps := monitor.GetActiveFlapsSummary()

	b, err := json.Marshal(activeFlaps)
	if err != nil {
		logger.Warn("Failed to marshal list to JSON", "error", err)
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
		if f.Prefix == prefix {
			var pathList []monitor.PathInfo
			if config.GlobalConf.KeepPathInfo {
				pathList = make([]monitor.PathInfo, 0)
				for v := range f.PathHistory.All() {
					pathList = append(pathList, *v)
				}
			}

			js, err := json.Marshal(struct {
				Prefix     string
				FirstSeen  int64
				RateSec    int
				TotalCount uint64
				Paths      []monitor.PathInfo
			}{
				f.Prefix.String(),
				f.FirstSeen,
				f.RateSec,
				f.TotalPathChanges,
				pathList,
			})
			if err != nil {
				w.WriteHeader(500)
				logger.Warn("Failed to marshal prefix to JSON", "error", err)
				_, _ = w.Write([]byte("Internal error"))
			} else {
				_, _ = w.Write(js)
			}
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

func getBgpSessions(w http.ResponseWriter, _ *http.Request) {
	info, err := monitor.GetSessionInfoJson()
	if err != nil {
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write([]byte(info))
}
