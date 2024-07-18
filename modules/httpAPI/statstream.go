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

	var channelClosedFirst = false
	go func() {
		// Listen for connection close
		<-r.Context().Done()
		clientMutex.Lock()
		if !channelClosedFirst {
			for i := 0; i < len(clients); i++ {
				if clients[i] == messageChan {
					clients[i] = clients[len(clients)-1]
					clients = clients[:len(clients)-1]
					break
				}
			}
		}
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
			channelClosedFirst = true
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
		clientMutex.Lock()
		tmp := clients[:0]
		for _, c := range clients {
			select {
			case c <- formatEventStreamMessage("u", string(m)):
			default:
				close(c)
				continue
			}
			tmp = append(tmp, c)
		}
		clients = tmp
		clientMutex.Unlock()
	}
}

func formatEventStreamMessage(eventName string, data string) string {
	return fmt.Sprintf("event: %s\ndata:%s\n\n", eventName, data)
}
