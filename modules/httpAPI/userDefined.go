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
		_, _ = w.Write([]byte(formatEventStreamMessage("e", "Invalid prefix")))
		flusher.Flush()
		return
	}

	if monitor.GetNumberOfUserDefinedMonitorClients() >= int(*maxUserDefinedMonitors) {
		_, _ = w.Write([]byte(formatEventStreamMessage("e", "Maximum number of user-defined tracked prefixes reached")))
		flusher.Flush()
		return
	}

	statisticChannel, err := monitor.NewUserDefinedMonitor(prefix)
	if err != nil {
		_, _ = w.Write([]byte(formatEventStreamMessage("e", err.Error())))
		return
	}

	defer func() {
		// Give the user time to potentially retrieve path statistics via the view paths page
		time.Sleep(2 * time.Second)
		monitor.RemoveUserDefinedMonitor(prefix, statisticChannel)
	}()

	_, _ = w.Write([]byte(formatEventStreamMessage("valid", "")))

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

			_, err = w.Write([]byte(formatEventStreamMessage("u", string(result))))
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
		w.WriteHeader(500)
		return
	}

	f, found := analyze.GetUserDefinedMonitorEvent(prefix)
	if !found {
		_, _ = w.Write([]byte("null"))
		return
	}

	pathList := make([]analyze.PathInfo, 0)
	for v := range f.PathHistory.All() {
		pathList = append(pathList, *v)
	}

	js, err := json.Marshal(struct {
		Prefix     string
		FirstSeen  int64
		RateSec    int
		TotalCount uint64
		Paths      []analyze.PathInfo
	}{
		f.Prefix.String(),
		f.FirstSeen,
		f.RateSec,
		f.TotalPathChanges,
		pathList,
	})
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(js)
}
