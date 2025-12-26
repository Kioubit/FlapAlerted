//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type RoaResponse struct {
	Metadata   RoaMetadata `json:"metadata"`
	RoaEntries []RoaEntry  `json:"roas"`
}

type RoaMetadata struct {
	Counts    int   `json:"counts"`
	Generated int64 `json:"generated"`
	Valid     int64 `json:"valid"`
}

type RoaEntry struct {
	Prefix    string `json:"prefix"`
	MaxLength int    `json:"maxLength"`
	ASN       string `json:"asn"`
}

func getActiveFlapsRoa(w http.ResponseWriter, _ *http.Request) {
	activeFlaps := monitor.GetActiveFlapsSummary()

	// Build ROA entries
	roaEntries := make([]RoaEntry, len(activeFlaps))
	for i, flap := range activeFlaps {
		// Determine maxLength based on IPv4 or IPv6
		maxLength := 32
		if strings.Contains(flap.Prefix, ":") {
			maxLength = 128
		}

		roaEntries[i] = RoaEntry{
			Prefix:    flap.Prefix,
			MaxLength: maxLength,
			ASN:       "0",
		}
	}

	// Generate timestamps
	currentTime := time.Now()
	validTime := currentTime.Add(1 * time.Hour)

	// Build response
	response := RoaResponse{
		Metadata: RoaMetadata{
			Counts:    len(activeFlaps),
			Generated: currentTime.Unix(),
			Valid:     validTime.Unix(),
		},
		RoaEntries: roaEntries,
	}

	b, err := json.Marshal(response)
	if err != nil {
		logger.Warn("Failed to marshal ROA data to JSON", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(b)
}
