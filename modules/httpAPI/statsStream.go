//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/monitor"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (m *Module) getStatisticStream(w http.ResponseWriter, r *http.Request) {
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

	messageChan := make(chan []byte, 40)
	m.clientMutex.Lock()
	m.clients[messageChan] = struct{}{}
	m.clientMutex.Unlock()

	defer func() {
		m.clientMutex.Lock()
		if _, ok := m.clients[messageChan]; ok {
			// If it was not already closed and deleted by streamServe
			delete(m.clients, messageChan)
			close(messageChan)
		}
		m.clientMutex.Unlock()
	}()

	oldStats := monitor.GetStats()
	for _, stat := range oldStats {
		j, err := json.Marshal(stat)
		if err != nil {
			continue
		}
		if err := writeFrame(formatEventStreamMessage("c", j)); err != nil {
			return
		}
	}

	capabilities, err := getCapsWithModHTTPJSON()
	if err != nil {
		capabilities = []byte("{}")
	}
	if err := writeFrame(formatEventStreamMessage("ready", capabilities)); err != nil {
		return
	}

	for {
		select {
		case data, ok := <-messageChan:
			if !ok {
				return
			}
			if err := writeFrame(data); err != nil {
				return
			}
		case <-r.Context().Done():
			// Listen for connection close
			return
		}
	}
}

func (m *Module) streamServe(ctx context.Context) {
	statChan := monitor.SubscribeToStats()
	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-statChan:
			if !ok {
				return
			}
			j, err := json.Marshal(s)
			if err != nil {
				continue
			}
			message := formatEventStreamMessage("u", j)

			m.clientMutex.Lock()
			for c := range m.clients {
				select {
				case c <- message:
				default:
					delete(m.clients, c)
					close(c)
				}
			}
			m.clientMutex.Unlock()
		}
	}
}
