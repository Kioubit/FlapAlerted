package bgp

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/table"
	"FlapAlerted/config"
	"context"
	"log/slog"
	"net"
	"os"
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

func StartBGP(bgpListenAddress string, pathChangeChan chan table.PathChange) {
	listener, err := net.Listen("tcp", bgpListenAddress)
	if err != nil {
		slog.Error("Failed to start BGP listener", "error", err)
		os.Exit(1)
	}
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)
	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("Failed to accept TCP connection", "error", err.Error())
			continue
		}
		handleConnection(conn, pathChangeChan)
	}
}

func handleConnection(conn net.Conn, pathChangeChan chan table.PathChange) {
	logger := slog.With("remote", conn.RemoteAddr())
	logger.Info("New connection")

	updateChannel := make(chan table.SessionUpdateMessage, 10000)
	ctx, cancel := context.WithCancelCause(context.Background())

	go func() {
		t := table.NewPrefixTable(pathChangeChan)
		table.ProcessUpdates(cancel, updateChannel, t)
	}()

	go func() {
		newSession := &common.LocalSession{
			DefaultAFI:     common.AFI4,
			AddPathEnabled: config.GlobalConf.UseAddPath,
			Asn:            config.GlobalConf.Asn,
			RouterID:       config.GlobalConf.RouterID,
		}
		err, wasEstablished := newBGPConnection(ctx, logger, conn, newSession, updateChannel)
		if err != nil {
			logger.Error("connection encountered an error", "error", err.Error())
		}
		_ = conn.Close()
		close(updateChannel)
		if wasEstablished {
			common.RemoveSession(conn)
		}
	}()

}
