package config

var GlobalConf UserConfig

type UserConfig struct {
	RouteChangeCounter  int64
	FlapPeriod          int64
	Asn                 int64
	KeepPathInfo        bool
	UseAddPath          bool
	RelevantAsnPosition int64
	NotifyOnce          bool
	Debug               bool
}
