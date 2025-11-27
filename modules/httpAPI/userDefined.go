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
		_, _ = w.Write([]byte(formatEventStreamMessage("e", "Invalid prefix")))
		flusher.Flush()
		return
	}

	if monitor.GetNumberOfUserDefinedMonitorClients() >= int(*maxUserDefinedMonitors) {
		_, _ = w.Write([]byte(formatEventStreamMessage("e", "Maximum number of user-defined tracked prefixes reached")))
		flusher.Flush()
		return
	}

	var statisticChannel chan monitor.UserDefinedMonitorStatistic

	go func() {
		// Listen for connection close
		<-r.Context().Done()
		// Give the user time to potentially retrieve path statistics via the view paths page
		time.Sleep(2 * time.Second)
		monitor.RemoveUserDefinedMonitor(prefix, statisticChannel)
	}()

	if statisticChannel, err = monitor.NewUserDefinedMonitor(prefix); err != nil {
		_, _ = w.Write([]byte(formatEventStreamMessage("e", err.Error())))
		return
	}

	_, _ = w.Write([]byte(formatEventStreamMessage("valid", "")))

	for {
		data, ok := <-statisticChannel
		if !ok {
			return
		}
		result, err := json.Marshal(data)
		if err != nil {
			return
		}

		_, _ = w.Write([]byte(formatEventStreamMessage("u", string(result))))
		flusher.Flush()
	}
}

func getUserDefinedStatistic(w http.ResponseWriter, r *http.Request) {
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		w.WriteHeader(500)
		return
	}

	f := monitor.GetUserDefinedMonitorEvent(prefix)
	if f == nil {
		_, _ = w.Write([]byte("null"))
		return
	}

	pathList := make([]monitor.PathInfo, 0)
	for v := range f.PathHistory.All() {
		pathList = append(pathList, *v)
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
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(js)
}
