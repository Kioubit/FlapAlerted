//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	"encoding/json"
	"net/http"
	"net/netip"
	"time"
)

func (m *Module) getUserDefinedStatisticStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		_, _ = w.Write(formatEventStreamMessage("e", "Invalid prefix"))
		return
	}

	if !m.reserveUserDefinedSlot() {
		_, _ = w.Write(formatEventStreamMessage("e", "Maximum number of user-defined tracked prefixes reached"))
		return
	}
	defer m.releaseUserDefinedSlot()

	statisticChannel, err := monitor.NewUserDefinedMonitor(prefix)
	if err != nil {
		_, _ = w.Write(formatEventStreamMessage("e", err.Error()))
		return
	}

	defer func() {
		// Give the user time to potentially retrieve path statistics via the view paths page
		time.Sleep(2 * time.Second)
		monitor.RemoveUserDefinedMonitor(prefix, statisticChannel)
	}()

	_, _ = w.Write(formatEventStreamMessage("valid", ""))
	flusher.Flush()

	for {
		select {
		case data, ok := <-statisticChannel:
			if !ok {
				return
			}
			result, err := json.Marshal(data)
			if err != nil {
				return
			}

			_, err = w.Write(formatEventStreamMessage("u", result))
			if err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			// Listen for connection close
			return
		}
	}
}

func (m *Module) reserveUserDefinedSlot() bool {
	for {
		cur := m.userDefinedCount.Load()
		if cur >= int32(*maxUserDefinedMonitors) {
			return false
		}
		if m.userDefinedCount.CompareAndSwap(cur, cur+1) {
			return true
		}
		// Concurrent update, retry
	}
}

func (m *Module) releaseUserDefinedSlot() {
	m.userDefinedCount.Add(-1)
}

func (m *Module) getUserDefinedStatistic(w http.ResponseWriter, r *http.Request) {
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		m.sendAsJSON(w, nil)
		return
	}

	f, found := analyze.GetUserDefinedMonitorEvent(prefix)
	if !found {
		m.sendAsJSON(w, nil)
		return
	}
	m.sendAsJSON(w, f)
}
