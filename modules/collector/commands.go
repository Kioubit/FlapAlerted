//go:build !disable_mod_collector

package collector

import (
	"FlapAlerted/monitor"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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

func processCommand(command string, cancel context.CancelFunc, logger *slog.Logger) (response string, err error) {
	var cmd string
	var args []string
	cmd, args, err = parseCommand(command)
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
	case "NOTIFY_ERROR":
		response = "OK"
		message := ""
		if len(args) > 1 {
			message = strings.Join(args[1:], " ")
		}
		logger.Warn("Collector error notification", "collector_message", message)
		if len(args) == 0 || args[0] != "false" {
			cancel()
		}
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
		return "", errors.New("failed to marshal list to JSON")
	}
	return string(b), nil
}

func getCapabilities() (string, error) {
	capabilities := monitor.GetCapabilities()
	b, err := json.Marshal(capabilities)
	if err != nil {
		return "", errors.New("failed to marshal capabilities to JSON")
	}
	return string(b), nil
}
