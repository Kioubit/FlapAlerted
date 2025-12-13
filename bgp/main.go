package bgp

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/notification"
	"FlapAlerted/bgp/table"
	"FlapAlerted/config"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
)

/*
- RFCs implemented:
- "A Border Gateway Protocol 4 (BGP-4)" https://datatracker.ietf.org/doc/html/rfc4271
- "Multiprotocol Extensions for BGP-4" https://datatracker.ietf.org/doc/html/rfc4760
- "Advertisement of Multiple Paths in BGP" https://datatracker.ietf.org/doc/html/rfc7911
- "BGP Support for Four-Octet Autonomous System (AS) Number Space" https://datatracker.ietf.org/doc/html/rfc6793
- "Capabilities Advertisement with BGP-4" https://datatracker.ietf.org/doc/html/rfc3392
- "Extended Message Support for BGP" https://datatracker.ietf.org/doc/html/rfc8654
- "Hostname Capability for BGP" https://datatracker.ietf.org/doc/html/draft-walton-bgp-hostname-capability-02
*/

func StartBGP(ctx context.Context, parentWg *sync.WaitGroup, bgpListenAddress string) (<-chan table.PathChange, error) {
	pathChangeChan := make(chan table.PathChange, 1000)
	listener, err := net.Listen("tcp", bgpListenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to start listener: %w", err)
	}
	parentWg.Go(func() {
		defer close(pathChangeChan)
		defer func() {
			_ = listener.Close()
		}()

		go func() {
			<-ctx.Done()
			_ = listener.Close()
		}()

		var wg sync.WaitGroup
		defer wg.Wait()
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					slog.Info("BGP listener stopped", "reason", ctx.Err())
					return
				default:
					slog.Warn("Failed to accept TCP connection", "error", err)
					continue
				}
			}
			wg.Go(func() {
				handleConnection(ctx, conn, pathChangeChan)
			})
		}
	})
	return pathChangeChan, nil
}

func handleConnection(parent context.Context, conn net.Conn, pathChangeChan chan table.PathChange) {
	defer func() {
		_ = conn.Close()
	}()
	logger := slog.With("remote", conn.RemoteAddr())
	logger.Info("New connection")

	ctx, cancel := context.WithCancelCause(context.WithoutCancel(parent))
	defer cancel(nil)
	stop := context.AfterFunc(parent, func() {
		cancel(notification.ErrAdministrativeShutdown)
	})
	defer stop()

	var wg sync.WaitGroup
	defer wg.Wait()

	updateChannel := make(chan table.SessionUpdateMessage, 10000)
	defer close(updateChannel) // Must be after wg.Wait() so it runs first

	session := &common.LocalSession{
		DefaultAFI:     common.AFI4,
		AddPathEnabled: config.GlobalConf.UseAddPath,
		Asn:            config.GlobalConf.Asn,
		OwnRouterID:    config.GlobalConf.RouterID,
	}

	err := newBGPConnection(ctx, logger, conn, session)
	if err != nil {
		if ctx.Err() != nil {
			logger.Warn("connection initiation canceled")
			return
		}
		logger.Error("connection encountered an error during session initiation", "error", err.Error())
		return
	}

	wg.Go(func() {
		t := table.NewPrefixTable(pathChangeChan, cancel)
		table.ProcessUpdates(cancel, updateChannel, t)
	})

	common.AddSession(conn, session)
	defer common.RemoveSession(conn)

	logger = logger.With("routerID", session.RemoteRouterID.String(), "hostname", session.RemoteHostname)

	err = handleEstablished(ctx, cancel, conn, logger, session, updateChannel)
	if err != nil {
		if !errors.Is(err, notification.ErrAdministrativeShutdown) {
			logger.Error("connection encountered an error", "error", err.Error())
		}
	}

}
