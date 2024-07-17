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
	clientMutex sync.RWMutex
	clients     = make([]chan string, 0)
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
	messageChan := make(chan string, 40)

	clientMutex.Lock()
	clients = append(clients, messageChan)
	clientMutex.Unlock()

	go func() {
		// Listen for connection close
		<-r.Context().Done()
		clientMutex.Lock()
		newClients := make([]chan string, 0)
		for _, c := range clients {
			if c == messageChan {
				continue
			}
			newClients = append(newClients, c)
		}
		clients = newClients
		close(messageChan)
		clientMutex.Unlock()
	}()

	oldStats := monitor.GetStats()
	for _, stat := range oldStats {
		m, err := json.Marshal(stat)
		if err != nil {
			continue
		}
		_, _ = w.Write([]byte(formatEventStreamMessage("u", string(m))))
	}
	flusher.Flush()

	for {
		data, ok := <-messageChan
		if !ok {
			return
		}
		_, _ = w.Write([]byte(data))
		flusher.Flush()
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
		clientMutex.RLock()
		for _, c := range clients {
			c <- formatEventStreamMessage("u", string(m))
		}
		clientMutex.RUnlock()
	}
}

func formatEventStreamMessage(eventName string, data string) string {
	return fmt.Sprintf("event: %s\ndata:%s\n\n", eventName, data)
}
