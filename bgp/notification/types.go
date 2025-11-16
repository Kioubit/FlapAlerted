package notification

import (
	"fmt"
	"log/slog"
	"strconv"
)

type Msg struct {
	ErrorCode    ErrorCode
	ErrorSubCode ErrorSubCode
	ErrorData    []byte
}

type ErrorCode uint8
type ErrorSubCode uint8

const (
	MessageHeaderError      ErrorCode = 1
	OpenMessageError        ErrorCode = 2
	UpdateMessageError      ErrorCode = 3
	HoldTimerExpiredError   ErrorCode = 4
	FiniteStateMachineError ErrorCode = 5
	Cease                   ErrorCode = 6
)

func (m Msg) Error() string {
	return fmt.Sprintf("BGP %s error (subcode=%d)",
		m.ErrorCode.String(), m.ErrorSubCode)
}

func (e ErrorCode) String() string {
	switch e {
	case MessageHeaderError:
		return "MessageHeaderError"
	case OpenMessageError:
		return "OpenMessageError"
	case UpdateMessageError:
		return "UpdateMessageError"
	case HoldTimerExpiredError:
		return "HoldTimerExpiredError"
	case FiniteStateMachineError:
		return "FiniteStateMachineError"
	case Cease:
		return "Cease"
	default:
		return "Unknown: " + strconv.Itoa(int(e))
	}
}

const (
	OpenBadPeerAS                    ErrorSubCode = 2
	OpenUnsupportedOptionalParameter ErrorSubCode = 4
)

func (m Msg) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("error_code", m.ErrorCode.String()),
		slog.Int("error_sub_code", int(m.ErrorSubCode)),
	)
}
