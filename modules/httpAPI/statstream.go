//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

var (
	clientMutex sync.Mutex
	clients     = make(map[chan []byte]struct{})
)

func getStatisticStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	messageChan := make(chan []byte, 40)

	clientMutex.Lock()
	clients[messageChan] = struct{}{}
	clientMutex.Unlock()

	defer func() {
		clientMutex.Lock()
		if _, ok := clients[messageChan]; ok {
			// If it was not already closed and deleted by streamServe
			delete(clients, messageChan)
			close(messageChan)
		}
		clientMutex.Unlock()
	}()

	oldStats := monitor.GetStats()
	for _, stat := range oldStats {
		m, err := json.Marshal(stat)
		if err != nil {
			continue
		}
		_, err = w.Write(formatEventStreamMessage("c", string(m)))
		if err != nil {
			return
		}
	}

	capabilities, err := getCapsWithModHttpJSON()
	if err != nil {
		capabilities = []byte("{}")
	}
	_, err = w.Write(formatEventStreamMessage("ready", string(capabilities)))
	if err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case data, ok := <-messageChan:
			if !ok {
				return
			}
			_, err := w.Write(data)
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

func streamServe() {
	statChan := monitor.SubscribeToStats()
	for {
		s := <-statChan
		m, err := json.Marshal(s)
		if err != nil {
			continue
		}
		clientMutex.Lock()
		for c := range clients {
			select {
			case c <- formatEventStreamMessage("u", string(m)):
			default:
				delete(clients, c)
				close(c)
				continue
			}
		}
		clientMutex.Unlock()
	}
}

func formatEventStreamMessage(eventName string, data string) []byte {
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventName, data))
}
