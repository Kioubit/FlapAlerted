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

var Version = ""

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})))
	_, _ = fmt.Fprintln(os.Stderr, "FlapAlerted", Version)
	monitor.SetVersion(Version)

	routeChangeCounter := flag.Uint("routeChangeCounter", 700, "Number of times a route path needs"+
		" to change to list a prefix. Use '0' to show all route changes.")
	asn := flag.Uint("asn", 0, "Your ASN number")
	overThresholdTarget := flag.Uint("overThresholdTarget", 10, "Number of consecutive intervals with rate at or above the routeChangeCounter to trigger an event")
	underThresholdTarget := flag.Uint("underThresholdTarget", 10, "Number of consecutive intervals with rate below routeChangeCounter to remove an event")
	routerID := flag.String("routerID", "0.0.0.51", "BGP Router ID for this program")
	noPathInfo := flag.Bool("noPathInfo", false, "Disable keeping path information")
	disableAddPath := flag.Bool("disableAddPath", false, "Disable BGP AddPath support. (Setting must be replicated in BGP daemon)")
	bgpListenAddress := flag.String("bgpListenAddress", ":1790", "Address to listen on for incoming BGP connections")
	enableDebug := flag.Bool("debug", false, "Enable debug mode (produces a lot of output)")

	flag.Parse()

	// Support environment variables
	flag.VisitAll(func(f *flag.Flag) {
		var env string
		env = os.Getenv("FA_" + strings.ToUpper(f.Name))
		if env == "" {
			env = os.Getenv("FA_" + f.Name)
		}
		if env != "" {
			err := flag.Set(f.Name, env)
			if err != nil {
				fmt.Println("Invalid value for the environment variable", "FA_"+strings.ToUpper(f.Name))
				os.Exit(1)
			}
		}
	})

	conf := config.UserConfig{}
	conf.RouteChangeCounter = int(*routeChangeCounter)
	conf.OverThresholdTarget = int(*overThresholdTarget)
	conf.UnderThresholdTarget = int(*underThresholdTarget)
	conf.Asn = uint32(*asn)
	conf.KeepPathInfo = !*noPathInfo
	conf.UseAddPath = !*disableAddPath
	conf.Debug = *enableDebug
	conf.BgpListenAddress = *bgpListenAddress

	if conf.Asn == 0 {
		fmt.Println("ASN value not specified")
		os.Exit(1)
	}

	if conf.RouteChangeCounter == 0 {
		conf.OverThresholdTarget = 0
		conf.UnderThresholdTarget = 0
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

	slog.Info("Started", "parameters", fmt.Sprintf(
		"Detecting flaps: trigger alert after %d consecutive 60s intervals with >= %d route changes; "+
			"end alert after %d consecutive 60s intervals with < %d route changes",
		conf.OverThresholdTarget, conf.RouteChangeCounter,
		conf.UnderThresholdTarget, conf.RouteChangeCounter))

	monitor.StartMonitoring(conf)
}
