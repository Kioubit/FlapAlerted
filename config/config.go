package config

import "net/netip"

var GlobalConf UserConfig

type UserConfig struct {
	RouteChangeCounter  int
	FlapPeriod          int
	MinimumAge          int
	Asn                 uint32
	KeepPathInfo        bool
	UseAddPath          bool
	RelevantAsnPosition int
	Debug               bool
	RouterID            netip.Addr
}
