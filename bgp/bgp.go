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
	"sync"
	"time"
)

func newBGPConnection(ctx context.Context, logger *slog.Logger, conn net.Conn, session *common.LocalSession) (err error) {
	stop := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stop()

	const ownHoldTime = 240
	err = conn.SetDeadline(time.Now().Add(ownHoldTime * time.Second))
	if err != nil {
		logger.Warn("Failed to set connection deadline", "error", err)
	}

	myCapabilities := []open.CapabilityOptionalParameter{
		{
			CapabilityCode:  open.CapabilityCodeFourByteASN,
			CapabilityValue: open.FourByteASNCapability{ASN: session.Asn},
		},
		{
			CapabilityCode: open.CapabilityCodeMultiProtocol,
			CapabilityValue: open.MultiProtocolCapability{
				AFI:  common.AFI4,
				SAFI: common.UNICAST,
			},
		},
		{
			CapabilityCode: open.CapabilityCodeMultiProtocol,
			CapabilityValue: open.MultiProtocolCapability{
				AFI:  common.AFI6,
				SAFI: common.UNICAST,
			},
		},
		{
			CapabilityCode:  open.CapabilityCodeExtendedMessage,
			CapabilityValue: open.ExtendedMessageCapability{},
		},
		{
			CapabilityCode: open.CapabilityCodeHostname,
			CapabilityValue: open.HostnameCapability{
				Hostname: "flapalerted",
			},
		},
		{
			CapabilityCode: open.CapabilityCodeExtendedNextHop,
			CapabilityValue: open.ExtendedNextHopCapabilityList{
				open.ExtendedNextHopCapability{
					AFI:        common.AFI4,
					SAFI:       common.UNICAST,
					NextHopAFI: common.AFI6,
				},
			},
		},
	}

	if session.AddPathEnabled {
		myCapabilities = append(myCapabilities, open.CapabilityOptionalParameter{
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
		})
	}

	openMessage, err := open.GetOpen(ownHoldTime, session.OwnRouterID, myCapabilities...)
	if err != nil {
		return fmt.Errorf("error marshalling OPEN message: %w", err)
	}
	_, err = conn.Write(openMessage)
	if err != nil {
		return fmt.Errorf("error writing OPEN message: %w", err)
	}

	// Read peer OPEN message
	msg, r, err := common.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading OPEN message from peer: %w", err)
	}
	if msg.Header.BgpType != common.MsgOpen {
		if msg.Header.BgpType == common.MsgNotification {
			if notificationMsg, err := notification.ParseMsgNotification(r); err == nil {
				return fmt.Errorf("notification during open: %w", notificationMsg)
			}
		} else {
			if nMsg, err := notification.GetNotification(notification.FiniteStateMachineError, 0, []byte{}); err == nil {
				_, _ = conn.Write(nMsg)
			}
		}
		return fmt.Errorf("unexpected message of type '%s', expected open", msg.Header.BgpType)
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
	var remoteASN uint32 = 0
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
				session.RemoteHostname = v.String()
			case open.ExtendedNextHopCapabilityList:
				for _, ec := range v {
					if ec.AFI == common.AFI4 && ec.NextHopAFI == common.AFI6 && ec.SAFI == common.UNICAST {
						session.HasExtendedNextHopV4 = true
					}
				}
			case open.ExtendedMessageCapability:
				session.HasExtendedMessages = true
			}
		}
	}
	peerHoldTime := msg.Body.(open.Msg).HoldTime.GetApplicableSeconds()
	if peerHoldTime > 900 || peerHoldTime == 1 || peerHoldTime == 2 {
		// Values 1 and 2 are disallowed by RFC, zero is allowed
		if nMsg, err := notification.GetNotification(notification.OpenMessageError, notification.OpenUnacceptableHoldTime, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return fmt.Errorf("unacceptable peer hold time of %d seconds", peerHoldTime)
	}
	slog.Debug("Peer hold time: %d seconds", peerHoldTime)

	applicableHoldTime := ownHoldTime
	if peerHoldTime < ownHoldTime {
		applicableHoldTime = peerHoldTime
	}
	session.ApplicableHoldTime = applicableHoldTime

	session.RemoteRouterID = msg.Body.(open.Msg).RouterID.ToNetAddr()

	if !hasFourByteAsn {
		if nMsg, err := notification.GetNotification(notification.OpenMessageError, notification.OpenUnsupportedOptionalParameter, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return fmt.Errorf("four byte ASNs not supported by peer")
	}
	if remoteASN != session.Asn {
		if nMsg, err := notification.GetNotification(notification.OpenMessageError, notification.OpenBadPeerAS, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return fmt.Errorf("remote ASN (%d) does not match the set asn (%d)", remoteASN, session.Asn)
	}

	if !hasMultiProtocolIPv4 && !hasMultiProtocolIPv6 {
		if nMsg, err := notification.GetNotification(notification.OpenMessageError, notification.OpenUnsupportedOptionalParameter, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return fmt.Errorf("multiprotocol capability is not supported by peer")
	}

	if session.AddPathEnabled && (!hasAddPathIPv6 || !hasAddPathIPv4) {
		if nMsg, err := notification.GetNotification(notification.OpenMessageError, notification.OpenUnsupportedOptionalParameter, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return fmt.Errorf("addPath is not supported by peer")
	}

	keepAliveBytes, _ := GetKeepAlive()
	_, err = conn.Write(keepAliveBytes)
	if err != nil {
		return fmt.Errorf("error writing keep alive message: %w", err)
	}

	msg, r, err = common.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading KEEPALIVE message from peer: %w", err)
	}
	if msg.Header.BgpType != common.MsgKeepAlive {
		if msg.Header.BgpType == common.MsgNotification {
			notificationMsg, err := notification.ParseMsgNotification(r)
			if err != nil {
				return fmt.Errorf("error parsing notification message: %w", err)
			}
			return fmt.Errorf("peer reported an error: %w", notificationMsg)
		}
		return fmt.Errorf("unexpected message of type '%s', expected keepalive", msg.Header.BgpType)
	}
	return nil
}

func handleEstablished(ctx context.Context, ctxCancel context.CancelCauseFunc, conn net.Conn, logger *slog.Logger, session *common.LocalSession, updateChannel chan table.SessionUpdateMessage) error {
	logger.Info("BGP session established")

	// From this point on the hold timer will manage the connection deadline
	err := conn.SetDeadline(time.Time{})
	if err != nil {
		logger.Warn("failed to reset connection deadline", "error", err)
	}

	keepAliveChan := make(chan struct{}, 1)
	defer close(keepAliveChan)
	var wg sync.WaitGroup
	defer wg.Wait()
	keepAliveHandler(ctx, ctxCancel, &wg, logger, keepAliveChan, conn, session.ApplicableHoldTime)

	err = handleMessages(ctx, logger, conn, session, updateChannel, keepAliveChan, session.HasExtendedMessages)
	if err != nil {
		// Give the receiver time to receive the notification before closing the connection
		defer time.Sleep(1 * time.Second)

		ctxCause := context.Cause(ctx)
		if ctxCause != nil {
			// Context has been canceled
			if errors.Is(ctxCause, notification.ErrImportLimit) {
				if nMsg, err := notification.GetNotification(notification.Cease, notification.CeaseMaxNumberOfPrefixes, []byte{}); err == nil {
					_, _ = conn.Write(nMsg)
				}
			} else if errors.Is(ctxCause, notification.ErrAdministrativeShutdown) {
				if nMsg, err := notification.GetNotification(notification.Cease, notification.CeaseAdministrativeShutdown, []byte{}); err == nil {
					_, _ = conn.Write(nMsg)
				}
			} else if errors.Is(ctxCause, notification.ErrHoldTimeExpired) {
				if nMsg, err := notification.GetNotification(notification.HoldTimerExpiredError, 0, []byte{}); err == nil {
					_, _ = conn.Write(nMsg)
				}
			}
			return ctxCause
		}
		// Context was not canceled, error in the function
		if nMsg, err := notification.GetNotification(notification.UpdateMessageError, notification.UpdateMessageErrorUnspecific, []byte{}); err == nil {
			_, _ = conn.Write(nMsg)
		}
		return err
	}
	logger.Info("BGP Connection closed")
	return nil
}

func keepAliveHandler(ctx context.Context, ctxCancel context.CancelCauseFunc, wg *sync.WaitGroup, logger *slog.Logger, in <-chan struct{}, conn net.Conn, holdTime int) {
	if holdTime == 0 {
		return
	}
	holdDuration := time.Duration(holdTime) * time.Second
	keepAliveInterval := holdDuration / 4
	updateThreshold := holdDuration / 10
	if updateThreshold < 2*time.Second {
		updateThreshold = 2 * time.Second
	}
	sleepTimer := time.NewTimer(updateThreshold)

	wg.Go(func() {
		ticker := time.NewTicker(keepAliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				// Ensure anything remaining (like bgp notification writes) terminates quickly
				err := conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
				if err != nil {
					_ = conn.Close()
				}
				// Stop any reads immediately
				err = conn.SetReadDeadline(time.Now())
				if err != nil {
					_ = conn.Close()
				}
				return
			case <-ticker.C:
			}
			keepAliveBytes, _ := GetKeepAlive()
			_, err := conn.Write(keepAliveBytes)
			if err != nil {
				logger.Debug("Error sending keepalive", "error", err)
				ctxCancel(err)
				return
			}
		}
	})
	wg.Go(func() {
		timer := time.NewTimer(holdDuration)
		defer timer.Stop()
		// time.after in a select statement cannot be used to avoid large amounts of channel allocations
		for {
			// No need to run this loop thousands of times per second
			sleepTimer.Reset(updateThreshold)
			select {
			case <-ctx.Done():
				return
			case <-sleepTimer.C:
			}
			select {
			case <-ctx.Done():
				return
			case _, ok := <-in:
				if !ok {
					return
				}
				// Safe to call without draining as of GO 1.23
				timer.Reset(holdDuration)
			case <-timer.C:
				ctxCancel(notification.ErrHoldTimeExpired)
				return
			}
		}
	})
}

func handleMessages(ctx context.Context, logger *slog.Logger, conn io.Reader, session *common.LocalSession, updateChannel chan table.SessionUpdateMessage, keepAliveChan chan<- struct{}, hasExtendedMessages bool) error {
	bufferSize := 4096
	if hasExtendedMessages {
		bufferSize = 4096 * 10
	}
	conn = bufio.NewReaderSize(conn, bufferSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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
			case keepAliveChan <- struct{}{}:
			default:
			}
		case common.MsgOpen:
			return errors.New("invalid state. Got OPEN message while the session was already established")
		case common.MsgUpdate:
			// Reset holdTimer as per RFC
			select {
			case keepAliveChan <- struct{}{}:
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
