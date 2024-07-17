package bgp

import (
	"FlapAlerted/bgp/update"
	"FlapAlerted/config"
	"log"
	"log/slog"
	"net"
)

// RFCs implemented
// "A Border Gateway Protocol 4 (BGP-4)" https://datatracker.ietf.org/doc/html/rfc4271
// "Multiprotocol Extensions for BGP-4" https://datatracker.ietf.org/doc/html/rfc4760
// "Advertisement of Multiple Paths in BGP" https://datatracker.ietf.org/doc/html/rfc7911
// "BGP Support for Four-Octet Autonomous System (AS) Number Space" https://datatracker.ietf.org/doc/html/rfc6793
// "Capabilities Advertisement with BGP-4" https://datatracker.ietf.org/doc/html/rfc3392

func StartBGP(updateChannel chan update.Msg) {
	listener, err := net.Listen("tcp", ":1790")
	if err != nil {
		log.Fatal("[FATAL]", err.Error())
	}
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)
	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("Failed to accept TCP connection", "error", err.Error())
		}

		logger := slog.With("remote", conn.RemoteAddr())
		logger.Info("New connection")

		go func() {
			err := newBGPConnection(logger, conn, update.AFI4, config.GlobalConf.UseAddPath, config.GlobalConf.Asn,
				config.GlobalConf.RouterID, updateChannel)
			if err != nil {
				logger.Error("connection encountered an error", "error", err.Error())
			}
			_ = conn.Close()
		}()
	}
}
