//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/netip"
	"strconv"
)

func (m *Module) mainPageHandler() http.Handler {
	html, _ := fs.Sub(dashboardContent, "www/dist")
	fileServer := http.FileServer(http.FS(html))

	withETag := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; object-src 'none'; form-action 'self'; style-src 'self' 'unsafe-inline'")
		w.Header().Set("ETag", m.eTag)
		w.Header().Set("Cache-Control", "public, max-age=900")

		if r.Header.Get("If-None-Match") == m.eTag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
	return withETag
}

func getCapsWithModHTTPJSON() ([]byte, error) {
	type modHTTPCaps struct {
		GageMaxValue       uint   `json:"gageMaxValue"`
		GageDisableDynamic bool   `json:"gageDisableDynamic"`
		MaxUserDefined     uint   `json:"maxUserDefined"`
		AsnExplorerURL     string `json:"asnExplorerURL"`
	}

	type fullCaps struct {
		monitor.Capabilities
		ModHTTP modHTTPCaps `json:"modHttp"`
	}

	return json.Marshal(fullCaps{
		Capabilities: monitor.GetCapabilities(),
		ModHTTP: modHTTPCaps{
			GageMaxValue:       *gageMaxValue,
			GageDisableDynamic: *gageDisableDynamic,
			MaxUserDefined:     *maxUserDefinedMonitors,
			AsnExplorerURL:     *explorerURLASN,
		},
	})
}

func (m *Module) getPrefix(w http.ResponseWriter, r *http.Request) {
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		m.sendAsJSON(w, nil)
		return
	}

	f, found := analyze.GetActiveFlapPrefix(prefix)
	if !found {
		m.sendAsJSON(w, nil)
		return
	}
	m.sendAsJSON(w, f)
}

func (m *Module) getPeer(w http.ResponseWriter, r *http.Request) {
	asn, err := strconv.ParseUint(r.URL.Query().Get("asn"), 10, 32)
	if err != nil {
		m.sendAsJSON(w, nil)
		return
	}

	p, found := analyze.GetActivePeer(uint32(asn))
	if !found {
		m.sendAsJSON(w, nil)
		return
	}
	m.sendAsJSON(w, p)
}

func (m *Module) getHistoricalPrefix(w http.ResponseWriter, r *http.Request) {
	timestamp := r.URL.Query().Get("timestamp")
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		m.sendAsJSON(w, nil)
		return
	}
	provider := monitor.GetHistoryProvider()
	if provider == nil {
		m.sendAsJSON(w, nil)
		return
	}

	var f *analyze.FlapEvent
	var eventKey monitor.HistoricalEventKey
	if timestamp == "" {
		f, eventKey, err = provider.GetHistoricalEventLatest(prefix)
	} else {
		var timestampInt int64
		timestampInt, err = strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid timestamp value"))
			return
		}
		eventKey = monitor.HistoricalEventKey{
			Prefix:    prefix,
			Timestamp: timestampInt,
		}
		f, err = provider.GetHistoricalEvent(eventKey)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Error getting history event"))
		return
	}

	if f == nil {
		m.sendAsJSON(w, nil)
		return
	}
	m.sendAsJSON(w, struct {
		Event    analyze.FlapEvent
		EventKey monitor.HistoricalEventKey
	}{*f, eventKey})
}

func (m *Module) getHistoricalList(w http.ResponseWriter, _ *http.Request) {
	provider := monitor.GetHistoryProvider()
	if provider == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("No history provider available"))
		return
	}
	list, err := provider.GetHistoricalEventList()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Retrieval of historical events failed"))
		return
	}
	m.sendAsJSON(w, list)
}

func (m *Module) getBgpSessions(w http.ResponseWriter, _ *http.Request) {
	info, err := monitor.GetSessionInfoJson()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(info))
}

func (m *Module) getASNExplorerURL(w http.ResponseWriter, _ *http.Request) {
	m.sendAsJSON(w, struct {
		ASNExplorerURL string `json:"asnExplorerUrl"`
	}{
		ASNExplorerURL: *explorerURLASN,
	})
}
