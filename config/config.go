package config

import "net/netip"

var GlobalConf UserConfig

type UserConfig struct {
	RouteChangeCounter   int
	OverThresholdTarget  int
	UnderThresholdTarget int
	Asn                  uint32
	KeepPathInfo         bool
	UseAddPath           bool
	Debug                bool
	RouterID             netip.Addr
	BgpListenAddress     string
}
