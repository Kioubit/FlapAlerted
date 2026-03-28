package analyze

import (
	"net/netip"
)

func copyEvent(src *FlapEvent) (event FlapEvent) {
	// Shallow copy of the struct
	event = *src
	// Copy the slice in the struct
	event.RateSecHistory = make([]int, len(src.RateSecHistory))
	copy(event.RateSecHistory, src.RateSecHistory)
	return
}

func copyEventIfTriggered(src *FlapEvent) (event FlapEvent, triggered bool) {
	if !src.hasTriggered {
		return
	}
	triggered = true
	event = copyEvent(src)
	return
}

func GetActiveFlapList() (active []FlapEvent, trackedCount int) {
	aFlap := make([]FlapEvent, 0)
	activeMapLock.RLock()
	defer activeMapLock.RUnlock()
	trackedCount = len(activeMap)
	for _, src := range activeMap {
		event, triggered := copyEventIfTriggered(src)
		if !triggered {
			continue
		}
		aFlap = append(aFlap, event)
	}
	return aFlap, trackedCount
}

func GetActiveFlapPrefix(prefix netip.Prefix) (FlapEvent, bool) {
	activeMapLock.RLock()
	defer activeMapLock.RUnlock()
	src, found := activeMap[prefix]
	if !found {
		return FlapEvent{}, false
	}
	f, triggered := copyEventIfTriggered(src)
	if !triggered {
		return FlapEvent{}, false
	}
	return f, true
}

// Peer update rate tracking

func copyPeerRate(src *PeerUpdateRate) (pr PeerUpdateRate) {
	// Shallow copy of the struct
	pr = *src
	// Copy the slice in the struct
	pr.RateSecHistory = make([]int, len(src.RateSecHistory))
	copy(pr.RateSecHistory, src.RateSecHistory)

	return
}

func copyPeerRateIfActive(src *PeerUpdateRate) (pr PeerUpdateRate, triggered bool) {
	if src.RateSec == -1 {
		return
	}
	triggered = true
	pr = copyPeerRate(src)
	return
}

func calculateAverageRate(pr *PeerUpdateRate) {
	if len(pr.RateSecHistory) == 0 {
		pr.RateSecAvg = -1
		return
	}
	sum := 0
	for _, rate := range pr.RateSecHistory {
		sum += rate
	}
	pr.RateSecAvg = float64(sum) / float64(maxRateHistory)
}

func GetPeerRates() []PeerUpdateRate {
	aRates := func() []PeerUpdateRate {
		activeMapLock.RLock()
		defer activeMapLock.RUnlock()
		aRates := make([]PeerUpdateRate, 0, len(activeMapPeer))
		for _, peer := range activeMapPeer {
			pr, triggered := copyPeerRateIfActive(peer)
			if !triggered {
				continue
			}
			aRates = append(aRates, pr)
		}
		return aRates
	}()

	for i := range aRates {
		calculateAverageRate(&aRates[i])
	}
	return aRates
}

func GetActivePeer(asn uint32) (PeerUpdateRate, bool) {
	f, triggered := func() (PeerUpdateRate, bool) {
		activeMapLock.RLock()
		defer activeMapLock.RUnlock()
		src, found := activeMapPeer[asn]
		if !found {
			return PeerUpdateRate{}, false
		}
		return copyPeerRateIfActive(src)
	}()

	if !triggered {
		return PeerUpdateRate{}, false
	}

	calculateAverageRate(&f)
	return f, true
}
