//go:build !disable_mod_collector

package collector

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

func parseCommand(input string) (cmd string, args []string, err error) {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		err = errors.New("empty command")
		return
	}
	cmd = strings.ToUpper(parts[0])
	args = parts[1:]
	return
}

func processCommand(command string) (response string, err error) {
	var cmd string
	cmd, _, err = parseCommand(command)
	if err != nil {
		return
	}

	switch cmd {
	case "PING":
		response = "PONG"
	case "ACTIVE_FLAPS":
		response, err = getActiveFlapJSON()
	case "AVERAGE_ROUTE_CHANGES_90":
		response = strconv.FormatFloat(monitor.GetAverageRouteChanges90(), 'f', 2, 64)
	case "CAPABILITIES":
		response, err = getCapabilities()
	case "INSTANCE":
		response = *collectorInstanceName
	case "VERSION":
		response = monitor.GetProgramVersion()
	default:
		err = errors.New("unknown command")
	}
	return
}

func getActiveFlapJSON() (string, error) {
	activeFlaps := monitor.GetActiveFlapsSummary()

	b, err := json.Marshal(activeFlaps)
	if err != nil {
		logger.Warn("Failed to marshal list to JSON", "error", err)
		return "", errors.New("failed to marshal list to JSON")
	}
	return string(b), nil
}

func getCapabilities() (string, error) {
	capabilities := monitor.GetCapabilities()
	b, err := json.Marshal(capabilities)
	if err != nil {
		logger.Warn("Failed to marshal capabilities to JSON", "error", err)
		return "", errors.New("failed to marshal capabilities to JSON")
	}
	return string(b), nil
}
