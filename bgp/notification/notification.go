package notification

import (
	"bytes"
	"encoding/binary"
	"io"
)

func ParseMsgNotification(r io.Reader) (msg Msg, err error) {
	if err := binary.Read(r, binary.BigEndian, &msg.ErrorCode); err != nil {
		return msg, err
	}
	if err := binary.Read(r, binary.BigEndian, &msg.ErrorSubCode); err != nil {
		return msg, err
	}
	var dataBuffer bytes.Buffer
	if _, err = io.Copy(&dataBuffer, r); err != nil {
		return msg, err
	}
	msg.ErrorData = dataBuffer.Bytes()
	return msg, nil
}
