package common

import (
	"net/netip"
)

type LocalSession struct {
	DefaultAFI           AFI
	AddPathEnabled       bool
	Asn                  uint32
	OwnRouterID          netip.Addr
	RemoteRouterID       netip.Addr
	RemoteHostname       string
	HasExtendedNextHopV4 bool
	HasExtendedMessages  bool
	ApplicableHoldTime   int
}
