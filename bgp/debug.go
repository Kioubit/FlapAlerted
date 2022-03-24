package bgp

import "fmt"

var GlobalDebug = false

func debugPrintln(data ...any) {
	if GlobalDebug {
		fmt.Println(data...)
	}
}

func debugPrintf(s string, data ...any) {
	if GlobalDebug {
		fmt.Printf(s, data...)
	}
}
