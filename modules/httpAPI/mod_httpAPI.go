//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	"crypto/subtle"
	"embed"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var moduleName = "mod_httpAPI"

//go:embed dashboard/*
var dashboardContent embed.FS

var (
	limitedHttpAPI         = flag.Bool("httpAPILimit", false, "Disable http API endpoints not needed for the user interface and activate basic scraping protection")
	apiKey                 = flag.String("httpAPIKey", "", "API key to access limited endpoints, when 'limitedHttpApi' is set. Empty to disable")
	httpAPIListenAddress   = flag.String("httpAPIListenAddress", ":8699", "Listen address for the HTTP API (TCP address like :8699 or Unix socket path)")
	gageMaxValue           = flag.Uint("httpGageMaxValue", 400, "HTTP dashboard Gage max value")
	gageDisableDynamic     = flag.Bool("httpGageDisableDynamic", false, "Disable dynamic Gage max value based on session count")
	maxUserDefinedMonitors = flag.Uint("httpMaxUserDefined", 5, "Maximum number of user-defined tracked prefixes. Use zero to disable")
)

type Module struct {
	name string
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) OnStart() bool {
	go startComplete()
	return false
}

func (m *Module) OnEvent(_ analyze.FlapEvent, _ bool) {}

func init() {
	monitor.RegisterModule(&Module{
		name: "mod_httpAPI",
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

func startComplete() {
	go streamServe()

	mux := http.NewServeMux()
	// --- Primary endpoints ---
	mux.Handle("/", mainPageHandler())
	mux.HandleFunc("/flaps/prefix", antiScrapeMiddleware(getPrefix))
	mux.HandleFunc("/flaps/statStream", getStatisticStream)
	mux.HandleFunc("/sessions", antiScrapeMiddleware(getBgpSessions))

	mux.HandleFunc("/flaps/historical/prefix", antiScrapeMiddleware(getHistoricalPrefix))
	mux.HandleFunc("/flaps/historical/list", antiScrapeMiddleware(getHistoricalList))

	if *maxUserDefinedMonitors != 0 {
		mux.HandleFunc("/userDefined/subscribe", getUserDefinedStatisticStream)
		mux.HandleFunc("/userDefined/prefix", antiScrapeMiddleware(getUserDefinedStatistic))
	}

	// --- Secondary endpoints ---
	mux.HandleFunc("/capabilities", requireAPIKeyWhenLimited(getCapabilities))
	mux.HandleFunc("/flaps/avgRouteChanges90", requireAPIKeyWhenLimited(getAvgRouteChanges))
	mux.HandleFunc("/flaps/active/compact", requireAPIKeyWhenLimited(getActiveFlaps))
	mux.HandleFunc("/flaps/active/roa", requireAPIKeyWhenLimited(getActiveFlapsRoa))
	mux.HandleFunc("/flaps/metrics/json", requireAPIKeyWhenLimited(metrics))
	mux.HandleFunc("/flaps/metrics/prometheus", requireAPIKeyWhenLimited(prometheus))

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
	slog.Info("Start HTTP server", "listen_address", *httpAPIListenAddress)
	err = s.Serve(listener)
	if err != nil {
		logger.Error("Error starting HTTP API server", "error", err)
	}
}

func requireAPIKeyWhenLimited(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !*limitedHttpAPI {
			next(w, r)
			return
		}
		if *apiKey == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		key := r.Header.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(key), []byte(*apiKey)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
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
