package bgp

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/notification"
	"FlapAlerted/bgp/table"
	"FlapAlerted/config"
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
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

func StartBGP(ctx context.Context, bgpListenAddress string) <-chan table.PathChange {
	pathChangeChan := make(chan table.PathChange, 1000)
	go func() {
		defer close(pathChangeChan)
		listener, err := net.Listen("tcp", bgpListenAddress)
		if err != nil {
			slog.Error("Failed to start BGP listener", "error", err)
			os.Exit(1)
		}
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
					slog.Error("Failed to accept TCP connection", "error", err.Error())
					continue
				}
			}
			go handleConnection(ctx, &wg, conn, pathChangeChan)
		}
	}()
	return pathChangeChan
}

func handleConnection(parent context.Context, wg *sync.WaitGroup, conn net.Conn, pathChangeChan chan table.PathChange) {
	logger := slog.With("remote", conn.RemoteAddr())
	logger.Info("New connection")

	updateChannel := make(chan table.SessionUpdateMessage, 10000)
	ctx, cancel := context.WithCancelCause(context.WithoutCancel(parent))
	stop := context.AfterFunc(parent, func() {
		cancel(notification.ErrAdministrativeShutdown)
	})
	defer stop()

	wg.Go(func() {
		t := table.NewPrefixTable(pathChangeChan, cancel)
		table.ProcessUpdates(cancel, updateChannel, t)
	})

	newSession := &common.LocalSession{
		DefaultAFI:     common.AFI4,
		AddPathEnabled: config.GlobalConf.UseAddPath,
		Asn:            config.GlobalConf.Asn,
		RouterID:       config.GlobalConf.RouterID,
	}
	err, wasEstablished := newBGPConnection(ctx, cancel, logger, conn, newSession, updateChannel)
	if err != nil {
		if !errors.Is(err, notification.ErrAdministrativeShutdown) {
			logger.Error("connection encountered an error", "error", err.Error())
		}
	}
	_ = conn.Close()
	close(updateChannel)
	if wasEstablished {
		common.RemoveSession(conn)
	}
}
