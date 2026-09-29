//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	"context"
	cryptorand "crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:generate npm --prefix www run build

//go:embed www/dist
var dashboardContent embed.FS

var (
	limitedHttpAPI         = flag.Bool("httpAPILimit", false, "Disable http API endpoints not needed for the user interface and activate basic scraping protection")
	apiKey                 = flag.String("httpAPIKey", "", "API key to access limited endpoints, when 'limitedHttpApi' is set. Empty to disable")
	httpAPIListenAddress   = flag.String("httpAPIListenAddress", ":8699", "Listen address for the HTTP API (TCP address like :8699 or Unix socket path)")
	gageMaxValue           = flag.Uint("httpGageMaxValue", 400, "HTTP dashboard Gage max value")
	gageDisableDynamic     = flag.Bool("httpGageDisableDynamic", false, "Disable dynamic Gage max value based on session count")
	maxUserDefinedMonitors = flag.Uint("httpMaxUserDefined", 5, "Maximum number of user-defined tracked prefixes. Use zero to disable")
	explorerURLASN         = flag.String("httpASNExplorerURL", "", "URL template for an external ASN lookup explorer. '{asn}' is replaced with the ASN. Empty to disable")
)

type Module struct {
	name   string
	logger *slog.Logger

	// eTag for static content
	eTag string
	// Statistics stream
	clientMutex sync.Mutex
	clients     map[chan []byte]struct{}
	// User defined monitors
	userDefinedCount atomic.Int32
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) OnStart(ctx context.Context, wg *sync.WaitGroup) bool {
	m.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})).With("module", m.name)
	b := make([]byte, 12)
	_, _ = cryptorand.Read(b)
	m.eTag = fmt.Sprintf(`"%s"`, base64.RawURLEncoding.EncodeToString(b))
	m.clients = make(map[chan []byte]struct{})

	wg.Go(func() {
		m.startHTTPServer(ctx)
	})

	wg.Go(func() {
		m.streamServe(ctx)
	})

	return false
}

func (m *Module) OnEvent(_ analyze.FlapEvent, _ bool) {}

func init() {
	monitor.RegisterModule(&Module{
		name: "mod_httpAPI",
	})
}

func (m *Module) startHTTPServer(ctx context.Context) {
	mux := http.NewServeMux()
	// --- Primary endpoints ---
	mux.Handle("/", m.mainPageHandler())
	mux.HandleFunc("/flaps/prefix", antiScrapeMiddleware(m.getPrefix))
	mux.HandleFunc("/peers/asn", antiScrapeMiddleware(m.getPeer))
	mux.HandleFunc("/flaps/statStream", m.getStatisticStream)
	mux.HandleFunc("/sessions", antiScrapeMiddleware(m.getBgpSessions))
	mux.HandleFunc("/config/asnExplorerURL", antiScrapeMiddleware(m.getASNExplorerURL))

	mux.HandleFunc("/flaps/historical/prefix", antiScrapeMiddleware(m.getHistoricalPrefix))
	mux.HandleFunc("/flaps/historical/list", antiScrapeMiddleware(m.getHistoricalList))

	if *maxUserDefinedMonitors != 0 {
		mux.HandleFunc("/userDefined/subscribe", m.getUserDefinedStatisticStream)
		mux.HandleFunc("/userDefined/prefix", antiScrapeMiddleware(m.getUserDefinedStatistic))
	}

	// --- Secondary endpoints ---
	mux.HandleFunc("/capabilities", requireAPIKeyWhenLimited(m.getCapabilities))
	mux.HandleFunc("/peers/active", requireAPIKeyWhenLimited(m.getActivePeers))
	mux.HandleFunc("/flaps/avgRouteChanges90", requireAPIKeyWhenLimited(getAvgRouteChanges))
	mux.HandleFunc("/flaps/active/compact", requireAPIKeyWhenLimited(m.getActiveFlaps))
	mux.HandleFunc("/flaps/active/roa", requireAPIKeyWhenLimited(m.getActiveFlapsRoa))
	mux.HandleFunc("/flaps/metrics/json", requireAPIKeyWhenLimited(m.getMetrics))
	mux.HandleFunc("/flaps/metrics/prometheus", requireAPIKeyWhenLimited(getPrometheus))
	mux.HandleFunc("/flaps/metrics/prometheus/activePeerRates", requireAPIKeyWhenLimited(prometheusActivePeerRates))

	ctx, cancel := context.WithCancel(ctx)
	s := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	shutdownDone := make(chan struct{})
	context.AfterFunc(ctx, func() {
		defer close(shutdownDone)

		shutdownCtx, cancelShutdownCtx := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdownCtx()

		if shutdownErr := s.Shutdown(shutdownCtx); shutdownErr != nil {
			m.logger.Error("Graceful shutdown failed, forcing close", "error", shutdownErr)
			_ = s.Close()
		}
	})

	defer func() {
		cancel()
		<-shutdownDone
	}()

	var listener net.Listener
	var err error
	if strings.HasPrefix(*httpAPIListenAddress, "/") {
		_ = os.Remove(*httpAPIListenAddress)
		listener, err = net.Listen("unix", *httpAPIListenAddress)
		if err != nil {
			m.logger.Error("Error creating Unix listener", "error", err)
			return
		}
	} else {
		listener, err = net.Listen("tcp", *httpAPIListenAddress)
		if err != nil {
			m.logger.Error("Error creating TCP listener", "error", err)
			return
		}
	}
	m.logger.Info("Start HTTP server", "listen_address", *httpAPIListenAddress)

	if err = s.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		m.logger.Error("Error running HTTP API server", "error", err)
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
