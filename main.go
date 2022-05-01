package main

import (
	_ "FlapAlertedPro/modules"
	"FlapAlertedPro/monitor"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

var Version = "2.3"

func main() {
	fmt.Println("FlapAlertedPro", Version, "by Kioubit.dn42")
	monitor.SetVersion(Version)

	var defaultPeriod = 30
	var defaultCounter = 230
	var defaultAsn = 0
	var keepPathInfo = false
	var doAddPath = false
	var doPerPeerState = false
	var doDebug = false
	var notifyOnce = false

	if len(os.Args) == 9 {
		var err error
		defaultCounter, err = strconv.Atoi(os.Args[1])
		checkError(err)
		defaultPeriod, err = strconv.Atoi(os.Args[2])
		checkError(err)
		defaultAsn, err = strconv.Atoi(os.Args[3])
		checkError(err)

		if checkIsInputBool(os.Args[4]) {
			keepPathInfo = os.Args[4] == "true"
		} else {
			fmt.Println("keepPathInfo must be either 'true' or 'false'")
			os.Exit(1)
		}

		if checkIsInputBool(os.Args[5]) {
			doAddPath = os.Args[5] == "true"
		} else {
			fmt.Println("addPath must be either 'true' or 'false'")
			os.Exit(1)
		}

		if checkIsInputBool(os.Args[6]) {
			doPerPeerState = os.Args[6] == "true"
		} else {
			fmt.Println("perPeerState must be either 'true' or 'false'")
			os.Exit(1)
		}

		if checkIsInputBool(os.Args[7]) {
			notifyOnce = os.Args[7] == "true"
		} else {
			fmt.Println("notifyOnce must be either 'true' or 'false'")
			os.Exit(1)
		}

		if notifyOnce {
			fmt.Println("Warning: You will only get one notification per event with this option. That way you will not know when the flap event has ended.")
		}

		if checkIsInputBool(os.Args[8]) {
			doDebug = os.Args[8] == "true"
		} else {
			fmt.Println("doDebug must be either 'true' or 'false'")
			os.Exit(1)
		}

		if doDebug {
			fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			fmt.Println("CAUTION: You have enabled debug mode. This will generate a _ton_ of debug messages.")
			fmt.Println("Exit the program if this is a mistake. Waiting 10 seconds...")
			fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			time.Sleep(10 * time.Second)
		}

	} else {
		fmt.Println("Required commandline args missing: routeChangeCounter, flapPeriod, asn, keepPathInfo, addPath, PerPeerState, notifyOnce, debug")
		fmt.Println("Refer to the documentation for more information.")
		os.Exit(1)
	}

	empty := true
	for _, m := range monitor.GetRegisteredModules() {
		fmt.Println("Enabled module:", m.Name)
		empty = false
	}
	if empty {
		fmt.Println("Error: No modules enabled during compilation!")
		fmt.Printf("It is recommended to use the included Makefile")
		os.Exit(1)
	}

	fmt.Println("Using the following parameters:")
	fmt.Println("Detecting a flap if the route to a prefix changes within", defaultPeriod, "seconds at least", defaultCounter, "time(s)")
	fmt.Println("ASN:", defaultAsn, "| Keep Path Info:", keepPathInfo, "| AddPath Capability:", doAddPath, "| Keep per-peer State:", doPerPeerState, "| Notify once:", notifyOnce, "| Debug:", doDebug)

	log.Println("Started")
	monitor.StartMonitoring(uint32(defaultAsn), int64(defaultPeriod), uint64(defaultCounter), doAddPath, doPerPeerState, doDebug, notifyOnce, keepPathInfo)
}

func checkIsInputBool(input string) bool {
	if input == "true" || input == "false" {
		return true
	}
	return false
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
