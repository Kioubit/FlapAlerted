package httpAPI

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/monitor"
	"encoding/json"
	"net/http"
	"net/netip"
	"time"
)

func getUserDefinedStatisticStream(w http.ResponseWriter, r *http.Request) {
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		w.WriteHeader(500)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	closeChan := make(chan struct{})

	go func() {
		// Listen for connection close
		<-r.Context().Done()
		close(closeChan)
		monitor.RemoveUserDefined(prefix)
	}()

	if err = monitor.NewUserDefined(prefix); err != nil {
		_, _ = w.Write([]byte(formatEventStreamMessage("e", err.Error())))
		return
	}

	for {
		select {
		case <-closeChan:
			return
		default:
		}
		result, err := json.Marshal(struct {
			Count    uint64
			Sessions int
		}{
			Count:    monitor.GetUserDefinedEventCount(prefix),
			Sessions: common.GetSessionCount(),
		})
		if err != nil {
			return
		}

		_, _ = w.Write([]byte(formatEventStreamMessage("u", string(result))))
		flusher.Flush()
		time.Sleep(5 * time.Second)
	}
}

func getUserDefinedStatistic(w http.ResponseWriter, r *http.Request) {
	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		w.WriteHeader(500)
		return
	}

	f := monitor.GetUserDefinedEvent(prefix)
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
		f.FirstSeen.Unix(),
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
