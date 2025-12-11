//go:build !disable_mod_collector

package collector

import (
	"FlapAlerted/monitor"
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var moduleName = "mod_collector"

var collectorInstanceName *string
var collectorEndpoint *string
var useTLS *bool

const (
	maxCommandLength     = 1024
	maxCommandsPerMinute = 15
)

func init() {
	collectorInstanceName = flag.String("collectorInstanceName", "", "Instance name for this instance to send to the flap collector")
	collectorEndpoint = flag.String("collectorEndpoint", "", "Flap collector TCP endpoint")
	useTLS = flag.Bool("collectorUseTLS", false, "Whether to use TLS to the endpoint")

	monitor.RegisterModule(&monitor.Module{
		Name:            moduleName,
		OnStartComplete: startComplete,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

func startComplete() {
	if *collectorEndpoint == "" || *collectorInstanceName == "" {
		return
	}
	go connectAndListen()
}

func connectAndListen() {
	for {
		conn, err := net.DialTimeout("tcp", *collectorEndpoint, 15*time.Second)
		if err != nil {
			logger.Error("failed to connect to collector", "endpoint", *collectorEndpoint, "error", err)
			time.Sleep(5 * time.Minute)
			continue
		}

		if *useTLS {
			tlsConn := tls.Server(conn, &tls.Config{})
			err = tlsConn.Handshake()
			if err != nil {
				logger.Error("TLS handshake failed", "error", err)
				_ = conn.Close()
				time.Sleep(5 * time.Minute)
				continue
			}
			conn = tlsConn
		}

		logger.Info("connected to collector", "endpoint", *collectorEndpoint)

		if err := handleConnection(conn); err != nil {
			logger.Error("connection error", "error", err)
		}

		_ = conn.Close()
		logger.Info("disconnected from collector, retrying in 30 seconds")
		time.Sleep(30 * time.Second)
	}
}

func handleConnection(conn net.Conn) (err error) {
	// Send Instance name
	_, err = fmt.Fprintf(conn, "HELLO %s\n", *collectorInstanceName)
	if err != nil {
		return err
	}
	// Send Version
	_, err = fmt.Fprintf(conn, "VERSION %s\n", monitor.GetProgramVersion())
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(conn)
	buf := make([]byte, maxCommandLength)
	scanner.Buffer(buf, maxCommandLength)

	writer := bufio.NewWriter(conn)

	commandCount := 0
	resetTime := time.Now().Add(time.Minute)

	for {
		// Set read deadline to detect connection issues
		err = conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		if err != nil {
			return
		}

		if time.Now().After(resetTime) {
			commandCount = 0
			resetTime = time.Now().Add(time.Minute)
		}

		if commandCount >= maxCommandsPerMinute {
			return errors.New("rate limit exceeded")
		}
		commandCount++

		if !scanner.Scan() {
			if err = scanner.Err(); err != nil {
				return fmt.Errorf("scanner error: %w", err)
			}
			return nil
		}

		command := strings.TrimSpace(scanner.Text())
		logger.Debug("received command", "command", command)

		response := processCommand(command)

		// Send response
		_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
		_, err = writer.WriteString(response + "\n")
		if err != nil {
			return fmt.Errorf("write error: %w", err)
		}

		err = writer.Flush()
		if err != nil {
			return fmt.Errorf("flush error: %w", err)
		}

		logger.Debug("sent response", "response", response)
	}
}

func processCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "ERROR: empty command"
	}

	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "PING":
		return "PONG"

	case "ACTIVE_FLAPS":
		return getActiveFlapJSON()

	case "AVERAGE_ROUTE_CHANGES_90":
		return strconv.FormatFloat(monitor.GetAverageRouteChanges90(), 'f', 2, 64)

	case "INSTANCE":
		return *collectorInstanceName

	case "VERSION":
		return monitor.GetProgramVersion()

	default:
		return "ERROR: received unknown command"
	}
}

func getActiveFlapJSON() string {
	activeFlaps := monitor.GetActiveFlapsSummary()

	b, err := json.Marshal(activeFlaps)
	if err != nil {
		logger.Warn("Failed to marshal list to JSON", "error", err)
		return "ERROR: failed to marshal list to JSON"
	}
	return string(b)
}
