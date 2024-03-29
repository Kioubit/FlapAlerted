package main

import (
	"FlapAlerted/config"
	_ "FlapAlerted/modules"
	"FlapAlerted/monitor"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var Version = "3.6"

func main() {
	fmt.Println("Kioubit FlapAlerted", Version)
	monitor.SetVersion(Version)
	conf := &config.UserConfig{}

	v := reflect.Indirect(reflect.ValueOf(conf))
	for i := 0; i < v.NumField(); i++ {
		if len(os.Args) != v.NumField()+1 {
			showUsage("invalid number of commandline arguments")
		}
		field := v.Field(i)
		fieldName := v.Type().Field(i).Name
		fieldTag := v.Type().Field(i).Tag
		overloadStringTag := fieldTag.Get("overloadString")
		switch field.Kind() {
		case reflect.Int64:
			input, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				if overloadStringTag == "true" {
					if os.Args[i+1] != "auto" {
						showUsage(fmt.Sprintf("The value entered for %s is not a number or 'auto'", fieldName))
					}
					input = -1
				} else {
					showUsage(fmt.Sprintf("The value entered for %s is not a number", fieldName))
				}
			}
			if !field.OverflowInt(int64(input)) {
				field.SetInt(int64(input))
			} else {
				showUsage(fmt.Sprintf("The value entered for %s is too high", fieldName))
			}
		case reflect.Bool:
			if !checkIsInputBool(os.Args[i+1]) {
				showUsage(fmt.Sprintf("The value entered for %s must be either 'true' or 'false'", fieldName))
			}
			input := os.Args[i+1] == "true"
			field.SetBool(input)
		}
	}

	if conf.RelevantAsnPosition == -1 {
		if conf.UseAddPath {
			conf.RelevantAsnPosition = 1
		} else {
			conf.RelevantAsnPosition = 0
		}
	}

	empty := true
	modules := monitor.GetRegisteredModules()
	for _, m := range modules {
		fmt.Println("Enabled module:", m.Name)
		empty = false
	}
	if empty {
		fmt.Println("Error: No modules enabled during compilation!")
		fmt.Printf("It is recommended to use the included Makefile")
		os.Exit(1)
	}

	if conf.Debug {
		fmt.Println("WARNING: Debug mode has been activated which will generate a lot of output")
		fmt.Println("Waiting for 10 seconds...")
		time.Sleep(10 * time.Second)
	}

	fmt.Println("Using the following parameters:")
	fmt.Println("Detecting a flap if the route to a prefix changes within", conf.FlapPeriod, "seconds at least", conf.RouteChangeCounter, "time(s)")
	fmt.Println("ASN:", conf.Asn, "| Keep Path Info:", conf.KeepPathInfo, "| AddPath Capability:", conf.UseAddPath, "| Relevant ASN Position:", conf.RelevantAsnPosition)

	log.Println("Started")
	monitor.StartMonitoring(*conf)
}

func checkIsInputBool(input string) bool {
	if input == "true" || input == "false" {
		return true
	}
	return false
}

func getArguments() []string {
	v := reflect.ValueOf(config.UserConfig{})
	args := make([]string, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		args[i] = v.Type().Field(i).Name
	}
	return args
}

func showUsage(reason string) {
	if reason != "" {
		fmt.Println("Error:", reason)
	}
	fmt.Println("Usage:", os.Args[0], "<"+strings.Join(getArguments(), `> <`)+`>`)
	fmt.Println("Refer to the documentation for the meaning of those arguments")
	os.Exit(1)
}
