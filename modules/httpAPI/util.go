//go:build !disable_mod_httpAPI

package httpAPI

import (
	"encoding/json"
	"net/http"
)

func (m *Module) sendAsJSON(w http.ResponseWriter, data any) {
	out, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		m.logger.Debug("Error encoding JSON", "error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func formatEventStreamMessage[T string | []byte](eventName string, data T) []byte {
	b := make([]byte, 0, len(eventName)+len(data)+16)

	b = append(b, "event: "...)
	b = append(b, eventName...)
	b = append(b, "\ndata: "...)
	b = append(b, data...)
	b = append(b, "\n\n"...)

	return b
}
