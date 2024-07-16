package main

import (
	"FlapAlerted/config"
	_ "FlapAlerted/modules"
	"FlapAlerted/monitor"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"
)

var Version = "3.8"

func main() {
	fmt.Println("FlapAlerted version", Version)
	monitor.SetVersion(Version)

	routeChangeCounter := flag.Int("routeChangeCounter", 0, "Number of times a route path needs to change to list a prefix")
	flapPeriod := flag.Int("period", 60, "Interval in seconds within which the routeChangeCounter value is evaluated")
	asn := flag.Int("asn", 0, "Your ASN number")
	noPathInfo := flag.Bool("noPathInfo", false, "Disable keeping path information. (Only disable if performance is a concern)")
	disableAddPath := flag.Bool("disableAddPath", false, "Disable BGP AddPath support. (Setting must be replicated in BGP daemon)")
	relevantAsnPosition := flag.Int("", -1, "The position of the last static ASN (and for which to keep separate state for)"+
		" in each path. If AddPath support has been enabled this value is '1', otherwise it is '0'. For special cases like route collectors the value may differ.")
	enableDebug := flag.Bool("debug", false, "Enable debug mode (produces a lot of output)")

	flag.Parse()

	conf := config.UserConfig{}
	conf.RouteChangeCounter = int64(*routeChangeCounter)
	conf.FlapPeriod = int64(*flapPeriod)
	conf.Asn = int64(*asn)
	conf.KeepPathInfo = !*noPathInfo
	conf.UseAddPath = !*disableAddPath
	conf.RelevantAsnPosition = int64(*relevantAsnPosition)
	if conf.RelevantAsnPosition == -1 {
		if conf.UseAddPath {
			conf.RelevantAsnPosition = 1
		} else {
			conf.RelevantAsnPosition = 0
		}
	}
	conf.Debug = *enableDebug

	if conf.Asn == 0 {
		fmt.Println("ASN value not specified")
		os.Exit(1)
	}

	modules := monitor.GetRegisteredModules()
	for _, m := range modules {
		fmt.Println("Enabled module:", m.Name)
	}

	if conf.Debug {
		fmt.Println("Debug mode has been activated which will generate a lot of output")
		fmt.Println("Waiting for 4 seconds...")
		time.Sleep(4 * time.Second)
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})))
	}

	fmt.Println("Using the following parameters:")
	fmt.Println("Detecting a flap if the route to a prefix changes within", conf.FlapPeriod, "seconds at least", conf.RouteChangeCounter, "time(s)")
	fmt.Println("ASN:", conf.Asn, "| Keep Path Info:", conf.KeepPathInfo, "| AddPath Capability:", conf.UseAddPath, "| Relevant ASN Position:", conf.RelevantAsnPosition)

	slog.Info("Started")
	monitor.StartMonitoring(conf)
}
