package main

import (
	"FlapAlerted/config"
	_ "FlapAlerted/modules"
	"FlapAlerted/monitor"
	"flag"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strings"
	"time"
)

var Version = "3.13"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})))
	_, _ = fmt.Fprintln(os.Stderr, "FlapAlerted", "version", Version)
	monitor.SetVersion(Version)

	routeChangeCounter := flag.Int("routeChangeCounter", 700, "Number of times a route path needs"+
		" to change to list a prefix. Use '0' to show all route changes.")
	flapPeriod := flag.Int("period", 60, "Interval in seconds within which the"+
		" routeChangeCounter value is evaluated. Higher values increase memory consumption.")
	asn := flag.Int("asn", 0, "Your ASN number")
	routerID := flag.String("routerID", "0.0.0.51", "BGP Router ID for this program")
	noPathInfo := flag.Bool("noPathInfo", false, "Disable keeping path information. (only disable if memory usage is a concern)")
	pathInfoDetectedOnly := flag.Bool("pathInfoDetectedOnly", false, "Keep path information only for detected prefixes (decreases memory usage)")
	disableAddPath := flag.Bool("disableAddPath", false, "Disable BGP AddPath support. (Setting must be replicated in BGP daemon)")
	relevantAsnPosition := flag.Int("asnPosition", -1, "The position of the last static ASN (and for which to keep separate state for)"+
		" in each path. Use of this parameter is required for special cases such as when connected to a route collector.")
	minimumAge := flag.Int("minimumAge", 540, "Minimum age in seconds a prefix must be active to be detected."+
		" Has no effect if the routeChangeCounter is set to zero")
	enableDebug := flag.Bool("debug", false, "Enable debug mode (produces a lot of output)")

	flag.Parse()

	conf := config.UserConfig{}
	conf.RouteChangeCounter = *routeChangeCounter
	conf.FlapPeriod = *flapPeriod
	conf.MinimumAge = *minimumAge
	conf.Asn = uint32(*asn)
	conf.KeepPathInfo = !*noPathInfo
	conf.KeepPathInfoDetectedOnly = *pathInfoDetectedOnly
	conf.UseAddPath = !*disableAddPath
	conf.RelevantAsnPosition = *relevantAsnPosition
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

	var err error
	conf.RouterID, err = netip.ParseAddr(*routerID)
	if err != nil {
		fmt.Println("Invalid Router ID:", err)
		os.Exit(1)
	}

	modules := monitor.GetRegisteredModuleNames()
	if len(modules) != 0 {
		slog.Info("Enabled", "modules", strings.Join(modules, ","))
	}

	if conf.Debug {
		fmt.Println("Debug mode has been activated which will generate a lot of output")
		fmt.Println("Waiting for 4 seconds...")
		time.Sleep(4 * time.Second)
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})))
	}

	slog.Info("Program started", "parameters", fmt.Sprintf(
		"Detecting a flap if the route to a prefix changes within %d seconds at least %d time(s)"+
			" and remains active for at least %d seconds", conf.FlapPeriod, conf.RouteChangeCounter, conf.MinimumAge))

	slog.Info("Started")
	monitor.StartMonitoring(conf)
}
