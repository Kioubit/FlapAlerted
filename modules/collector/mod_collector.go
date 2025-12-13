//go:build !disable_mod_collector

package collector

import (
	"FlapAlerted/monitor"
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"
)

var moduleName = "mod_collector"

var collectorInstanceName *string
var collectorEndpoint *string
var useTLS *bool

func init() {
	collectorInstanceName = flag.String("collectorInstanceName", "", "Instance name for this instance to send to the flap collector")
	collectorEndpoint = flag.String("collectorEndpoint", "", "Flap collector TCP endpoint")
	useTLS = flag.Bool("collectorUseTLS", false, "Whether to use TLS to the collector endpoint")

	monitor.RegisterModule(&monitor.Module{
		Name:            moduleName,
		OnStartComplete: startComplete,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

const (
	maxCommandLength     = 1024
	maxCommandsPerMinute = 15
)

func startComplete() {
	if *collectorEndpoint == "" || *collectorInstanceName == "" {
		if *collectorEndpoint != "" {
			logger.Error("Collector endpoint specified but no instance name given!")
		}
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	connectAndListen(ctx, cancel)
}

func connectAndListen(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		conn, err := net.DialTimeout("tcp", *collectorEndpoint, 15*time.Second)
		if err != nil {
			logger.Error("Failed to connect to collector. Retry in 5 minutes", "endpoint", *collectorEndpoint, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Minute):
				continue
			}
		}

		if *useTLS {
			tlsConn := tls.Server(conn, &tls.Config{})
			err = tlsConn.Handshake()
			if err != nil {
				logger.Error("TLS handshake failed", "error", err)
				_ = conn.Close()
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Minute):
					continue
				}
			}
			conn = tlsConn
		}

		logger.Info("Connected to collector", "endpoint", *collectorEndpoint)

		if err := handleConnection(ctx, cancel, conn); err != nil {
			if ctx.Err() == nil {
				logger.Error("connection error", "error", err)
			}
		}

		_ = conn.Close()
		if ctx.Err() != nil {
			logger.Error("Disconnected from collector. Won't reconnect.")
			return
		}
		logger.Warn("Disconnected from collector, reconnecting in 30 seconds")

		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func handleConnection(ctx context.Context, cancel context.CancelFunc, conn net.Conn) (err error) {
	stop := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stop()

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
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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
			time.Sleep(5 * time.Minute)
			return errors.New("rate limit exceeded")
		}
		commandCount++

		if !scanner.Scan() {
			if err = scanner.Err(); err != nil {
				return fmt.Errorf("scanner error: %w", err)
			}
			return nil
		}

		command := scanner.Text()
		logger.Debug("received command", "command", command)

		response, err := processCommand(command, cancel)
		if err != nil {
			response = fmt.Sprintf("ERROR:%s", err.Error())
		}
		response = strings.ReplaceAll(response, "\n", "")

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
