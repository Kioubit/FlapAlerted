package analyze

import (
	"encoding/json"
	"net/netip"
)

type FlapEvent struct {
	// ===== Core data =====
	Prefix           netip.Prefix
	PathHistory      *PathTracker
	TotalPathChanges uint64

	// ===== Rate calculation =====
	RateSecHistory    []int
	lastIntervalCount uint64
	RateSec           int

	// ===== State tracking =====
	FirstSeen           int64
	overThresholdCount  int
	underThresholdCount int
	hasTriggered        bool
}

type FlapEventNotification struct {
	Event   FlapEvent
	IsStart bool
}

type FlapEventNoPaths FlapEvent

func (fe *FlapEventNoPaths) MarshalJSON() ([]byte, error) {
	type Alias FlapEvent

	return json.Marshal(&struct {
		*Alias
		PathHistory PathTrackerSummary `json:"PathHistory"`
	}{
		Alias:       (*Alias)(fe),
		PathHistory: PathTrackerSummary{fe.PathHistory},
	})
}
