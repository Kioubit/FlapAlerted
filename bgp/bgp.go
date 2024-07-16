package bgp

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/open"
	"FlapAlerted/bgp/update"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"time"
)

func newBGPConnection(logger *slog.Logger, conn net.Conn, defaultAFI update.AFI, addPathEnabled bool, routerID netip.Addr, updateChannel chan update.Msg) error {
	const ownHoldTime = 240
	err := conn.SetDeadline(time.Now().Add(ownHoldTime * time.Second))
	if err != nil {
		logger.Warn("Failed to set connection deadline", "error", err)
	}

	openMessage, err := open.GetOpen(ownHoldTime, routerID,
		open.CapabilityOptionalParameter{
			CapabilityCode:  open.CapabilityCodeFourByteASN,
			CapabilityValue: open.FourByteASNCapability{ASN: 4242423914},
		},
		open.CapabilityOptionalParameter{
			CapabilityCode: open.CapabilityCodeMultiProtocol,
			CapabilityValue: open.MultiProtocolCapability{
				AFI:      update.AFI4,
				Reserved: 0,
				SAFI:     update.UNICAST,
			},
		},
		open.CapabilityOptionalParameter{
			CapabilityCode: open.CapabilityCodeMultiProtocol,
			CapabilityValue: open.MultiProtocolCapability{
				AFI:      update.AFI6,
				Reserved: 0,
				SAFI:     update.UNICAST,
			},
		},
		open.CapabilityOptionalParameter{
			CapabilityCode: open.CapabilityCodeAddPath,
			CapabilityValue: open.AddPathCapabilityList{
				open.AddPathCapability{
					AFI:  update.AFI4,
					SAFI: update.UNICAST,
					TXRX: open.ReceiveOnly,
				},
				open.AddPathCapability{
					AFI:  update.AFI6,
					SAFI: update.UNICAST,
					TXRX: open.ReceiveOnly,
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error marshalling OPEN message %w", err)
	}
	_, err = conn.Write(openMessage)
	if err != nil {
		return fmt.Errorf("error writing OPEN message %w", err)
	}

	// Read peer OPEN message
	msg, r, err := common.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading OPEN message from peer %w", err)
	}
	if msg.Header.BgpType != common.MsgOpen {
		return fmt.Errorf("unexpected message of type '%d', expected open", msg.Header.BgpType)
	}

	msg.Body, err = open.ParseMsgOpen(r)
	if err != nil {
		return fmt.Errorf("error parsing peer OPEN message %w", err)
	}

	hasMultiProtocolIPv4 := false
	hasMultiProtocolIPv6 := false
	hasAddPathIPv4 := false
	hasAddPathIPv6 := false
	hasFourByteAsn := false
	for _, p := range msg.Body.(open.Msg).OptionalParameters {
		for _, t := range p.ParameterValue.(open.CapabilityList).List {
			switch v := t.CapabilityValue.(type) {
			case open.FourByteASNCapability:
				hasFourByteAsn = true
			case open.AddPathCapabilityList:
				for _, ac := range v {
					if ac.AFI == update.AFI4 {
						if ac.TXRX != open.ReceiveOnly {
							hasAddPathIPv4 = true
						}
					} else if ac.AFI == update.AFI6 {
						if ac.TXRX != open.ReceiveOnly {
							hasAddPathIPv6 = true
						}
					}
				}
			case open.MultiProtocolCapability:
				if v.SAFI != update.UNICAST {
					continue
				}
				if v.AFI == update.AFI4 {
					hasMultiProtocolIPv4 = true
				} else if v.AFI == update.AFI6 {
					hasMultiProtocolIPv6 = true
				}
			}
		}
	}

	peerHoldTime := msg.Body.(open.Msg).HoldTime.GetApplicableSeconds()
	applicableHoldTime := ownHoldTime
	if peerHoldTime < ownHoldTime {
		applicableHoldTime = peerHoldTime
	}
	applicableHoldTime = min(applicableHoldTime, 300)

	remoteRouterID := msg.Body.(open.Msg).RouterID

	if !hasFourByteAsn {
		return fmt.Errorf("four byte ASNs not supported by peer")
	}
	if !hasMultiProtocolIPv4 && !hasMultiProtocolIPv6 {
		return fmt.Errorf("multiprotocol capbility is not supported by peer")
	}

	if addPathEnabled && (!hasAddPathIPv6 || !hasAddPathIPv4) {
		return fmt.Errorf("addPath is not supported by peer")
	}

	keepAliveBytes, _ := GetKeepAlive()
	_, err = conn.Write(keepAliveBytes)
	if err != nil {
		return fmt.Errorf("error writing keep alive message %w", err)
	}

	msg, _, err = common.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading KEEPALIVE message from peer %w", err)
	}
	if msg.Header.BgpType != common.MsgKeepAlive {
		return fmt.Errorf("unexpected message of type '%d', expected keepalive", msg.Header.BgpType)
	}

	logger.Info("BGP session established", "routerID", remoteRouterID)

	// From this point on the hold timer will manage the connection deadline
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		logger.Warn("failed to reset connection deadline", "error", err)
	}

	keepAliveChan := make(chan bool, 1)
	defer close(keepAliveChan)
	keepAliveHandler(logger, keepAliveChan, conn, applicableHoldTime)

	err = handleIncoming(logger, conn, defaultAFI, addPathEnabled, updateChannel, keepAliveChan)
	if err != nil {
		return err
	}
	return nil
}

func keepAliveHandler(logger *slog.Logger, in chan bool, conn net.Conn, holdTime int) {
	if holdTime == 0 {
		return
	}
	go func() {
		for {
			time.Sleep(time.Duration(holdTime/4) * time.Second)
			keepAliveBytes, _ := GetKeepAlive()
			_, err := conn.Write(keepAliveBytes)
			if err != nil {
				logger.Debug("Error sending keepalive", "error", err)
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-time.After(time.Duration(holdTime) * time.Second):
				logger.Warn("hold time expired")
				_ = conn.Close()
				return
			case <-in:

			}
		}
	}()
}

func handleIncoming(logger *slog.Logger, conn io.Reader, defaultAFI update.AFI, addPathEnabled bool, updateChannel chan update.Msg, keepAliveChan chan bool) error {
	for {
		msg, r, err := common.ReadMessage(conn)
		if err != nil {
			return err
		}
		switch msg.Header.BgpType {
		case common.MsgNotification:
			return errors.New("bgp notification")
		case common.MsgKeepAlive:
			logger.Debug("Received keepalive message")
			keepAliveChan <- true
		case common.MsgOpen:
			return errors.New("invalid state. Got OPEN message while the session was already established")
		case common.MsgUpdate:
			msg.Body, err = update.ParseMsgUpdate(r, defaultAFI, addPathEnabled)
			if err != nil {
				return fmt.Errorf("failed parsing UPDATE message %w", err)
			}
			select {
			case updateChannel <- msg.Body.(update.Msg):
			default:
				// Cannot keep up
			}

		}

		// Discard any unread bytes
		_, err = io.Copy(io.Discard, r)
		if err != nil {
			return err
		}
	}
}
