//go:build !disable_mod_collector

package collector

import (
	"FlapAlerted/analyze"
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

var (
	collectorInstanceName = flag.String("collectorInstanceName", "", "Instance name for this instance to send to the flap collector")
	collectorEndpoint     = flag.String("collectorEndpoint", "", "Flap collector TCP endpoint")
	useTLS                = flag.Bool("collectorUseTLS", false, "Whether to use TLS to the collector endpoint")
)

type Module struct {
	name   string
	logger *slog.Logger
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) OnStart() bool {
	if *collectorEndpoint == "" && *collectorInstanceName == "" {
		return false
	}
	m.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", m.Name())
	if *collectorInstanceName == "" {
		m.logger.Error("Collector endpoint specified but no instance name given!")
	}

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		m.connectAndListen(ctx, cancel)
	}()

	return false
}

func (m *Module) OnEvent(_ analyze.FlapEvent, _ bool) {}

func init() {
	monitor.RegisterModule(&Module{
		name: "mod_collector",
	})
}

const (
	maxCommandLength     = 1024
	maxCommandsPerMinute = 15
)

func (m *Module) connectAndListen(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		conn, err := net.DialTimeout("tcp", *collectorEndpoint, 15*time.Second)
		if err != nil {
			m.logger.Error("Failed to connect to collector. Retry in 5 minutes", "endpoint", *collectorEndpoint, "error", err)
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
				m.logger.Error("TLS handshake failed", "error", err)
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

		m.logger.Info("Connected to collector", "endpoint", *collectorEndpoint)

		if err := handleConnection(ctx, cancel, m.logger, conn); err != nil {
			if ctx.Err() == nil {
				m.logger.Error("connection error", "error", err)
			}
		}

		_ = conn.Close()
		if ctx.Err() != nil {
			m.logger.Error("Disconnected from collector. Won't reconnect.")
			return
		}
		m.logger.Warn("Disconnected from collector, reconnecting in 30 seconds")

		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}
	}
}

func handleConnection(ctx context.Context, cancel context.CancelFunc, logger *slog.Logger, conn net.Conn) (err error) {
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

		response, err := processCommand(command, cancel, logger)
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
