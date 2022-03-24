package main

import (
	_ "FlapAlertedPro/modules"
	"FlapAlertedPro/monitor"
	"fmt"
	"os"
	"strconv"
)

var Version = "0.2"

func main() {
	fmt.Println("FlapAlerted Pro", Version, "by Kioubit.dn42")

	var defaultPeriod = 30
	var defaultCounter = 230
	var defaultAsn = 0

	if len(os.Args) == 4 {
		var err error
		defaultCounter, err = strconv.Atoi(os.Args[1])
		checkError(err)
		defaultPeriod, err = strconv.Atoi(os.Args[2])
		checkError(err)
		defaultAsn, err = strconv.Atoi(os.Args[3])
		checkError(err)
		fmt.Println("Using custom parameters", defaultCounter, defaultPeriod)
	} else {
		fmt.Println("Required commandline args missing: routeChangeCounter, flapPeriod,asn")
		os.Exit(1)
	}

	empty := true
	for _, m := range monitor.GetRegisteredModules() {
		fmt.Println("Enabled module:", m.Name)
		empty = false
	}
	if empty {
		fmt.Println("Error: No modules enabled during compilation!")
		os.Exit(1)
	}

	fmt.Println("Started")
	monitor.StartMonitoring(uint32(defaultAsn), int64(defaultPeriod), uint64(defaultCounter))
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
