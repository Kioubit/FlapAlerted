package config

import "net/netip"

var GlobalConf UserConfig

type UserConfig struct {
	RouteChangeCounter       int
	OverThresholdTarget      int
	UnderThresholdTarget     int
	ExpiryRouteChangeCounter int
	Asn                      uint32
	ImportLimit              uint32
	MaxPathHistory           int
	UseAddPath               bool
	Debug                    bool
	RouterID                 netip.Addr
	BgpListenAddress         string
}
