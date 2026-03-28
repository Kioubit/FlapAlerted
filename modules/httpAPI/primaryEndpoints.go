package httpAPI

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/netip"
	"strconv"
)

var eTag = ""

func init() {
	b := make([]byte, 12)
	_, _ = cryptorand.Read(b)
	eTag = fmt.Sprintf(`"%s"`, base64.RawURLEncoding.EncodeToString(b))
}

func mainPageHandler() http.Handler {
	html, _ := fs.Sub(dashboardContent, "dashboard")
	fileServer := http.FileServer(http.FS(html))

	withETag := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; form-action 'self'; object-src 'none'; style-src 'self' 'unsafe-inline'")
		w.Header().Set("ETag", eTag)
		w.Header().Set("Cache-Control", "public, max-age=900")

		if r.Header.Get("If-None-Match") == eTag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
	return withETag
}

func getCapsWithModHttpJSON() ([]byte, error) {
	caps := monitor.GetCapabilities()
	type ModHttpCaps struct {
		GageMaxValue         uint   `json:"gageMaxValue"`
		GageDisableDynamic   bool   `json:"gageDisableDynamic"`
		MaxUserDefined       uint   `json:"maxUserDefined"`
		ExplorerURLPrefixASN string `json:"explorerUrlPrefixASN"`
	}

	fullCaps := struct {
		monitor.Capabilities
		ModHttpCaps ModHttpCaps `json:"modHttp"`
	}{
		Capabilities: caps,
		ModHttpCaps: ModHttpCaps{
			GageMaxValue:         *gageMaxValue,
			GageDisableDynamic:   *gageDisableDynamic,
			MaxUserDefined:       *maxUserDefinedMonitors,
			ExplorerURLPrefixASN: *explorerURLPrefixASN,
		},
	}
	return json.Marshal(fullCaps)
}

func getPrefix(w http.ResponseWriter, r *http.Request) {
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		_, _ = w.Write([]byte("null"))
		return
	}

	f, found := analyze.GetActiveFlapPrefix(prefix)
	if !found {
		_, _ = w.Write([]byte("null"))
		return
	}
	_ = json.NewEncoder(w).Encode(f)
}

func getPeer(w http.ResponseWriter, r *http.Request) {
	asn, err := strconv.ParseUint(r.URL.Query().Get("asn"), 10, 32)
	if err != nil {
		_, _ = w.Write([]byte("null"))
		return
	}

	p, found := analyze.GetActivePeer(uint32(asn))
	if !found {
		_, _ = w.Write([]byte("null"))
		return
	}
	_ = json.NewEncoder(w).Encode(p)
}

func getHistoricalPrefix(w http.ResponseWriter, r *http.Request) {
	timestamp := r.URL.Query().Get("timestamp")
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid prefix"))
		return
	}
	provider := monitor.GetHistoryProvider()
	if provider == nil {
		_, _ = w.Write([]byte("null"))
		return
	}

	var f *analyze.FlapEvent
	var meta monitor.HistoricalEventMeta
	if timestamp == "" {
		f, meta, err = provider.GetHistoricalEventLatest(prefix)
	} else {
		var timestampInt int64
		timestampInt, err = strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid timestamp value"))
			return
		}
		meta = monitor.HistoricalEventMeta{
			Prefix:    prefix,
			Timestamp: timestampInt,
		}
		f, err = provider.GetHistoricalEvent(meta)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Error getting history event"))
		return
	}

	if f == nil {
		_, _ = w.Write([]byte("null"))
		return
	}
	_ = json.NewEncoder(w).Encode(struct {
		Event analyze.FlapEvent
		Meta  monitor.HistoricalEventMeta
	}{*f, meta})
}

func getHistoricalList(w http.ResponseWriter, _ *http.Request) {
	provider := monitor.GetHistoryProvider()
	if provider == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("A history provider module needs to be enabled and configured for this functionality"))
		return
	}
	list, err := provider.GetHistoricalEventList()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Retrieval of historical events failed"))
		return
	}
	_ = json.NewEncoder(w).Encode(list)
}

func getBgpSessions(w http.ResponseWriter, _ *http.Request) {
	info, err := monitor.GetSessionInfoJson()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte(info))
}
