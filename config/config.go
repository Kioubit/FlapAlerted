package config

var GlobalConf UserConfig

type UserConfig struct {
	RouteChangeCounter  int
	FlapPeriod          int64
	Asn                 uint32
	KeepPathInfo        bool
	UseAddPath          bool
	RelevantAsnPosition int
	Debug               bool
}
