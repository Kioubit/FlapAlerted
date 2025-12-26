//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"net/http"
	"net/netip"
	"time"
)

func getUserDefinedStatisticStream(w http.ResponseWriter, r *http.Request) {
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

	if monitor.GetNumberOfUserDefinedMonitorClients() >= int(*maxUserDefinedMonitors) {
		_, _ = w.Write(formatEventStreamMessage("e", "Maximum number of user-defined tracked prefixes reached"))
		return
	}

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

			_, err = w.Write(formatEventStreamMessage("u", string(result)))
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

func getUserDefinedStatistic(w http.ResponseWriter, r *http.Request) {
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		_, _ = w.Write([]byte("null"))
		return
	}

	js, err := flapToJSON(prefix, true)
	if err != nil {
		_, _ = w.Write([]byte("null"))
		return
	}
	_, _ = w.Write(js)
}
