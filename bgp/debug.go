package bgp

import (
	"FlapAlertedPro/config"
	"fmt"
)

func debugPrintln(data ...any) {
	if config.GlobalConf.Debug {
		fmt.Println(data...)
	}
}

func debugPrintf(s string, data ...any) {
	if config.GlobalConf.Debug {
		fmt.Printf(s, data...)
	}
}
