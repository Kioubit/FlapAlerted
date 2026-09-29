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
	rc := http.NewResponseController(w)
	writeFrame := func(msg []byte) error {
		if err := r.Context().Err(); err != nil {
			return err
		}

		if err := rc.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return err
		}

		if _, err := w.Write(msg); err != nil {
			return err
		}

		if err := rc.Flush(); err != nil {
			return err
		}

		return rc.SetWriteDeadline(time.Time{})
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	prefix, err := netip.ParsePrefix(r.URL.Query().Get("prefix"))
	if err != nil {
		_ = writeFrame(formatEventStreamMessage("e", "Invalid prefix"))
		return
	}
	prefix = prefix.Masked()

	if !m.reserveUserDefinedSlot() {
		_ = writeFrame(formatEventStreamMessage("e", "Maximum number of user-defined tracked prefixes reached"))
		return
	}
	defer m.releaseUserDefinedSlot()

	statisticChannel, err := monitor.NewUserDefinedMonitor(prefix)
	if err != nil {
		_ = writeFrame(formatEventStreamMessage("e", err.Error()))
		return
	}

	defer func() {
		// Give the user time to potentially retrieve path statistics via the view paths page
		time.Sleep(2 * time.Second)
		monitor.RemoveUserDefinedMonitor(prefix, statisticChannel)
	}()

	if err = writeFrame(formatEventStreamMessage("valid", "")); err != nil {
		return
	}

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

			if err = writeFrame(formatEventStreamMessage("u", result)); err != nil {
				return
			}
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
	prefix = prefix.Masked()

	f, found := analyze.GetUserDefinedMonitorEvent(prefix)
	if !found {
		m.sendAsJSON(w, nil)
		return
	}
	m.sendAsJSON(w, f)
}
