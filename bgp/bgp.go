package bgp

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/notification"
	"FlapAlerted/bgp/open"
	"FlapAlerted/bgp/table"
	"FlapAlerted/bgp/update"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"
)

func newBGPConnection(ctx context.Context, logger *slog.Logger, conn net.Conn, session *common.LocalSession, updateChannel chan table.SessionUpdateMessage) (err error, wasEstablished bool) {
	const ownHoldTime = 240
	err = conn.SetDeadline(time.Now().Add(ownHoldTime * time.Second))
	if err != nil {
		logger.Warn("Failed to set connection deadline", "error", err)
	}

	openMessage, err := open.GetOpen(ownHoldTime, session.RouterID,
		open.CapabilityOptionalParameter{
			CapabilityCode:  open.CapabilityCodeFourByteASN,
			CapabilityValue: open.FourByteASNCapability{ASN: session.Asn},
		},
		open.CapabilityOptionalParameter{
			CapabilityCode: open.CapabilityCodeMultiProtocol,
			CapabilityValue: open.MultiProtocolCapability{
				AFI:  common.AFI4,
				SAFI: common.UNICAST,
			},
		},
		open.CapabilityOptionalParameter{
			CapabilityCode: open.CapabilityCodeMultiProtocol,
			CapabilityValue: open.MultiProtocolCapability{
				AFI:  common.AFI6,
				SAFI: common.UNICAST,
			},
		},
		open.CapabilityOptionalParameter{
			CapabilityCode: open.CapabilityCodeAddPath,
			CapabilityValue: open.AddPathCapabilityList{
				open.AddPathCapability{
					AFI:  common.AFI4,
					SAFI: common.UNICAST,
					TXRX: open.ReceiveOnly,
				},
				open.AddPathCapability{
					AFI:  common.AFI6,
					SAFI: common.UNICAST,
					TXRX: open.ReceiveOnly,
				},
			},
		},
		open.CapabilityOptionalParameter{
			CapabilityCode:  open.CapabilityCodeExtendedMessage,
			CapabilityValue: open.ExtendedMessageCapability{},
		},
		open.CapabilityOptionalParameter{
			CapabilityCode: open.CapabilityCodeHostname,
			CapabilityValue: open.HostnameCapability{
				Hostname: "flapalerted",
			},
		},
		open.CapabilityOptionalParameter{
			CapabilityCode: open.CapabilityCodeExtendedNextHop,
			CapabilityValue: open.ExtendedNextHopCapabilityList{
				open.ExtendedNextHopCapability{
					AFI:        common.AFI4,
					SAFI:       common.UNICAST,
					NextHopAFI: common.AFI6,
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error marshalling OPEN message %w", err), false
	}
	_, err = conn.Write(openMessage)
	if err != nil {
		return fmt.Errorf("error writing OPEN message %w", err), false
	}

	// Read peer OPEN message
	msg, r, err := common.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading OPEN message from peer %w", err), false
	}
	if msg.Header.BgpType != common.MsgOpen {
		if msg.Header.BgpType == common.MsgNotification {
			if notificationMsg, err := notification.ParseMsgNotification(r); err == nil {
				return fmt.Errorf("notification during open: %w", notificationMsg), false
			}
		} else {
			if nMsg, err := notification.GetNotification(notification.FiniteStateMachineError, 0, []byte{}); err == nil {
				_, _ = conn.Write(nMsg)
			}
		}
		return fmt.Errorf("unexpected message of type '%s', expected open", msg.Header.BgpType), false
	}

	msg.Body, err = open.ParseMsgOpen(r)
	if err != nil {
		return fmt.Errorf("error parsing peer OPEN message %w", err), false
	}

	hasMultiProtocolIPv4 := false
	hasMultiProtocolIPv6 := false
	hasAddPathIPv4 := false
	hasAddPathIPv6 := false
	hasFourByteAsn := false
	hasExtendedNextHopV4 := false
	hasExtendedMessages := false
	var remoteASN uint32 = 0
	var remoteHostname = ""
	for _, p := range msg.Body.(open.Msg).OptionalParameters {
		for _, t := range p.ParameterValue.(open.CapabilityList).List {
			switch v := t.CapabilityValue.(type) {
			case open.FourByteASNCapability:
				remoteASN = v.ASN
				hasFourByteAsn = true
			case open.AddPathCapabilityList:
				for _, ac := range v {
					switch ac.AFI {
					case common.AFI4:
						if ac.TXRX != open.ReceiveOnly {
							hasAddPathIPv4 = true
						}
					case common.AFI6:
						if ac.TXRX != open.ReceiveOnly {
							hasAddPathIPv6 = true
						}
					}
				}
			case open.MultiProtocolCapability:
				if v.SAFI != common.UNICAST {
					continue
				}
				switch v.AFI {
				case common.AFI4:
					hasMultiProtocolIPv4 = true
				case common.AFI6:
					hasMultiProtocolIPv6 = true
				}
			case open.HostnameCapability:
				remoteHostname = v.String()
				logger = logger.With("hostname", remoteHostname)
			case open.ExtendedNextHopCapabilityList:
				for _, ec := range v {
					if ec.AFI == common.AFI4 && ec.NextHopAFI == common.AFI6 && ec.SAFI == common.UNICAST {
						hasExtendedNextHopV4 = true
					}
				}
			case open.ExtendedMessageCapability:
				hasExtendedMessages = true
			}
		}
	}
	session.HasExtendedNextHopV4 = hasExtendedNextHopV4
	peerHoldTime := msg.Body.(open.Msg).HoldTime.GetApplicableSeconds()
	applicableHoldTime := ownHoldTime
	if peerHoldTime < ownHoldTime {
		applicableHoldTime = peerHoldTime
	}
	applicableHoldTime = min(applicableHoldTime, 300)

	remoteRouterID := msg.Body.(open.Msg).RouterID

	if !hasFourByteAsn {
		if nMsg, err := notification.GetNotification(notification.OpenMessageError, notification.OpenUnsupportedOptionalParameter, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return fmt.Errorf("four byte ASNs not supported by peer"), false
	}
	if remoteASN != session.Asn {
		if nMsg, err := notification.GetNotification(notification.OpenMessageError, notification.OpenBadPeerAS, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return fmt.Errorf("remote ASN (%d) does not match the set asn (%d)", remoteASN, session.Asn), false
	}

	if !hasMultiProtocolIPv4 && !hasMultiProtocolIPv6 {
		if nMsg, err := notification.GetNotification(notification.OpenMessageError, notification.OpenUnsupportedOptionalParameter, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return fmt.Errorf("multiprotocol capbility is not supported by peer"), false
	}

	if session.AddPathEnabled && (!hasAddPathIPv6 || !hasAddPathIPv4) {
		if nMsg, err := notification.GetNotification(notification.OpenMessageError, notification.OpenUnsupportedOptionalParameter, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return fmt.Errorf("addPath is not supported by peer"), false
	}

	keepAliveBytes, _ := GetKeepAlive()
	_, err = conn.Write(keepAliveBytes)
	if err != nil {
		return fmt.Errorf("error writing keep alive message %w", err), false
	}

	msg, r, err = common.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading KEEPALIVE message from peer %w", err), false
	}
	if msg.Header.BgpType != common.MsgKeepAlive {
		if msg.Header.BgpType == common.MsgNotification {
			notificationMsg, err := notification.ParseMsgNotification(r)
			if err != nil {
				return fmt.Errorf("error parsing notification message %w", err), false
			}
			return fmt.Errorf("peer reported an error %s", notificationMsg), false
		} else {
			return fmt.Errorf("unexpected message of type '%s', expected keepalive", msg.Header.BgpType), false
		}
	}

	logger.Info("BGP session established", "routerID", remoteRouterID)
	common.AddSession(conn, remoteRouterID.String(), remoteHostname, session)

	// From this point on the hold timer will manage the connection deadline
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		logger.Warn("failed to reset connection deadline", "error", err)
	}

	keepAliveChan := make(chan bool, 1)
	defer close(keepAliveChan)
	keepAliveHandler(logger, keepAliveChan, conn, applicableHoldTime)

	err = handleIncoming(ctx, logger, conn, session, updateChannel, keepAliveChan, hasExtendedMessages)
	if err != nil {
		if errors.Is(err, &table.ImportLimitError{}) {
			if nMsg, err := notification.GetNotification(notification.Cease, notification.CeaseMaxNumberOfPrefixes, []byte{}); err == nil {
				_, _ = conn.Write(nMsg)
			}
		} else {
			if nMsg, err := notification.GetNotification(notification.UpdateMessageError, 0, []byte{}); err == nil {
				_, _ = conn.Write(nMsg)
			}
		}
		// Give the receiver time to receive the notification before closing the connection
		time.Sleep(1 * time.Second)
		return err, true
	}
	logger.Info("BGP Connection closed", "routerID", remoteRouterID)
	return nil, true
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
		holdTimeRemaining := holdTime
		// time.after in a select statement cannot be used to avoid large amounts of channel allocations
		for {
			time.Sleep(2 * time.Second)
			holdTimeRemaining -= 2
			select {
			case _, ok := <-in:
				if !ok {
					return
				}
				holdTimeRemaining = holdTime
			default:
			}
			if holdTimeRemaining <= 0 {
				logger.Warn("hold time expired")
				_ = conn.Close()
				return
			}
		}
	}()
}

func handleIncoming(ctx context.Context, logger *slog.Logger, conn io.Reader, session *common.LocalSession, updateChannel chan table.SessionUpdateMessage, keepAliveChan chan bool, hasExtendedMessages bool) error {
	bufferSize := 4096
	if hasExtendedMessages {
		bufferSize = 4096 * 10
	}
	conn = bufio.NewReaderSize(conn, bufferSize)
	for {
		select {
		case <-ctx.Done():
			err := context.Cause(ctx)
			return err
		default:
		}

		msg, r, err := common.ReadMessage(conn)
		if err != nil {
			return err
		}
		switch msg.Header.BgpType {
		case common.MsgNotification:
			notificationMsg, err := notification.ParseMsgNotification(r)
			if err != nil {
				return fmt.Errorf("failed parsing NOTIFICATION message %w", err)
			}
			logger.Debug("BGP notification", "message", notificationMsg)
			if notificationMsg.ErrorCode != notification.Cease {
				return notificationMsg
			}
			return nil
		case common.MsgKeepAlive:
			logger.Debug("Received keepalive message")
			select {
			case keepAliveChan <- true:
			default:
			}
		case common.MsgOpen:
			return errors.New("invalid state. Got OPEN message while the session was already established")
		case common.MsgUpdate:
			// Reset holdTimer as per RFC
			select {
			case keepAliveChan <- true:
			default:
			}
			msg.Body, err = update.ParseMsgUpdate(r, session.DefaultAFI, session.AddPathEnabled)
			if err != nil {
				return fmt.Errorf("failed parsing UPDATE message %w", err)
			}
			updateChannel <- table.SessionUpdateMessage{
				Msg:     msg.Body.(update.Msg),
				Session: session,
			}
		}

		// Discard any unread bytes
		_, err = io.Copy(io.Discard, r)
		if err != nil {
			return err
		}
	}
}
