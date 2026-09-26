//go:build !disable_mod_httpAPI

package httpAPI

import (
	"FlapAlerted/monitor"
	"net/http"
	"strings"
	"time"
)

type roaResponse struct {
	Metadata   roaMetadata `json:"metadata"`
	RoaEntries []roaEntry  `json:"roas"`
}

type roaMetadata struct {
	Counts    int   `json:"counts"`
	Generated int64 `json:"generated"`
	Valid     int64 `json:"valid"`
}

type roaEntry struct {
	Prefix    string `json:"prefix"`
	MaxLength int    `json:"maxLength"`
	ASN       string `json:"asn"`
}

func (m *Module) getActiveFlapsRoa(w http.ResponseWriter, _ *http.Request) {
	activeFlaps := monitor.GetActiveFlapsSummary()

	// Build ROA entries
	roaEntries := make([]roaEntry, len(activeFlaps))
	for i, flap := range activeFlaps {
		// Determine maxLength based on IPv4 or IPv6
		maxLength := 32
		if strings.Contains(flap.Prefix, ":") {
			maxLength = 128
		}

		roaEntries[i] = roaEntry{
			Prefix:    flap.Prefix,
			MaxLength: maxLength,
			ASN:       "0",
		}
	}

	// Generate timestamps
	currentTime := time.Now()
	validTime := currentTime.Add(1 * time.Hour)

	// Build response
	response := roaResponse{
		Metadata: roaMetadata{
			Counts:    len(activeFlaps),
			Generated: currentTime.Unix(),
			Valid:     validTime.Unix(),
		},
		RoaEntries: roaEntries,
	}

	m.sendAsJSON(w, response)
}
