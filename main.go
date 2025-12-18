package main

import (
	"FlapAlerted/config"
	_ "FlapAlerted/modules"
	"FlapAlerted/monitor"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var Version = ""

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})))
	_, _ = fmt.Fprintln(os.Stderr, "FlapAlerted", Version)
	monitor.SetProgramVersion(Version)

	// Flags
	var (
		asn                      = flag.Uint("asn", 0, "Your ASN number")
		overThresholdTarget      = flag.Uint("overThresholdTarget", 10, "Number of consecutive minutes with route change rate at or above the 'routeChangeCounter' to trigger an event")
		underThresholdTarget     = flag.Uint("underThresholdTarget", 15, "Number of consecutive minutes with route change rate below 'expiryRouteChangeCounter' to remove an event")
		routeChangeCounter       = flag.Uint("routeChangeCounter", 600, "Minimum change per minute threshold to detect a flap. Use '0' to show all route changes.")
		expiryRouteChangeCounter = flag.Uint("expiryRouteChangeCounter", 0, "Minimum change per minute threshold to keep detected flaps. Defaults to the same value as 'routeChangeCounter'.")
		routerID                 = flag.String("routerID", "0.0.0.51", "BGP router ID for this program")
		noPathInfo               = flag.Bool("noPathInfo", false, "Disable keeping path information")
		disableAddPath           = flag.Bool("disableAddPath", false, "Disable BGP AddPath support. (Setting must be replicated in BGP daemon)")
		bgpListenAddress         = flag.String("bgpListenAddress", ":1790", "Address to listen on for incoming BGP connections")
		enableDebug              = flag.Bool("debug", false, "Enable debug mode (produces a lot of output)")
		importLimitThousands     = flag.Uint("importLimitThousands", 10000, "Maximum number of allowed routes per session in thousands")
	)

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
	conf.ExpiryRouteChangeCounter = int(*expiryRouteChangeCounter)
	conf.Asn = uint32(*asn)
	conf.KeepPathInfo = !*noPathInfo
	conf.UseAddPath = !*disableAddPath
	conf.Debug = *enableDebug
	conf.BgpListenAddress = *bgpListenAddress
	conf.ImportLimit = uint32(*importLimitThousands * 1000)

	if conf.Asn == 0 {
		fmt.Println("ASN value not specified")
		os.Exit(1)
	}

	if conf.RouteChangeCounter == 0 {
		conf.OverThresholdTarget = 0
		conf.UnderThresholdTarget = 0
	} else if conf.OverThresholdTarget == 0 {
		conf.UnderThresholdTarget = 1
	}

	if conf.ExpiryRouteChangeCounter == 0 {
		conf.ExpiryRouteChangeCounter = conf.RouteChangeCounter
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

	var parameterString string
	if conf.RouteChangeCounter == 0 {
		parameterString = "Trigger an alert for all route changes. Remove entries after 60s of inactivity."
	} else {
		parameterString = fmt.Sprintf(
			"Trigger an alert after %d consecutive 60s intervals with > %d route changes; "+
				"end alert after %d consecutive 60s intervals with <= %d route changes",
			conf.OverThresholdTarget, conf.RouteChangeCounter,
			conf.UnderThresholdTarget, conf.ExpiryRouteChangeCounter)
	}

	slog.Info("Started", "parameters", parameterString)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err = monitor.StartMonitoring(ctx, conf)
	if err != nil && !errors.Is(err, context.Canceled) {
		slog.Info("Program stopped", "reason", err)
		os.Exit(1)
	}
	slog.Info("Program stopped")
}
