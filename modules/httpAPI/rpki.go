//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type RpkiMetadata struct {
	Counts    int   `json:"counts"`
	Generated int64 `json:"generated"`
	Valid     int64 `json:"valid"`
}

type RoaEntry struct {
	Prefix    string `json:"prefix"`
	MaxLength int    `json:"maxLength"`
	ASN       string `json:"asn"`
}

type RpkiResponse struct {
	Metadata RpkiMetadata `json:"metadata"`
	Roas     []RoaEntry   `json:"roas"`
}

func getActiveFlapsRpki(w http.ResponseWriter, r *http.Request) {
	activeFlaps := monitor.GetActiveFlapsSummary()

	// Build ROA entries
	roas := make([]RoaEntry, len(activeFlaps))
	for i, flap := range activeFlaps {
		// Determine maxLength based on IPv4 or IPv6
		maxLength := 32
		if strings.Contains(flap.Prefix, ":") {
			maxLength = 128
		}

		roas[i] = RoaEntry{
			Prefix:    flap.Prefix,
			MaxLength: maxLength,
			ASN:       "AS0",
		}
	}

	// Generate timestamps
	currentTime := time.Now().Unix()
	validTime := currentTime + 3600 // +1 hour

	// Build response
	response := RpkiResponse{
		Metadata: RpkiMetadata{
			Counts:    len(activeFlaps),
			Generated: currentTime,
			Valid:     validTime,
		},
		Roas: roas,
	}

	b, err := json.Marshal(response)
	if err != nil {
		logger.Warn("Failed to marshal list to JSON", "error", err)
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(b)
}
