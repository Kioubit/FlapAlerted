package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

type BgpMessage struct {
	Header BgpHeader
	Body   bgpBody
}

type bgpBody interface {
	MarshalBinary() ([]byte, error)
}

func (message BgpMessage) MarshalBinary() ([]byte, error) {
	// Body
	bodyBytes, err := message.Body.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// Header
	message.Header.Length = uint16(len(bodyBytes)) + 19
	var bHeader bytes.Buffer
	if err := binary.Write(&bHeader, binary.BigEndian, message.Header.Marker); err != nil {
		return nil, err
	}
	if err := binary.Write(&bHeader, binary.BigEndian, message.Header.Length); err != nil {
		return nil, err
	}
	if err := binary.Write(&bHeader, binary.BigEndian, message.Header.BgpType); err != nil {
		return nil, err
	}

	return append(bHeader.Bytes(), bodyBytes...), nil
}

type BgpHeader struct {
	Marker  [16]byte
	Length  uint16
	BgpType msgType
}

type msgType uint8

const (
	MsgOpen         msgType = 0x01
	MsgUpdate       msgType = 0x02
	MsgNotification msgType = 0x03
	MsgKeepAlive    msgType = 0x04
)

var bgpMarker = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

func GetHeader(t msgType) BgpHeader {
	return BgpHeader{
		Marker:  [16]byte(bgpMarker),
		BgpType: t,
	}
}

func ReadMessage(r io.Reader) (msg BgpMessage, bodyReader io.Reader, err error) {
	msg.Header = BgpHeader{}
	err = binary.Read(r, binary.BigEndian, &msg.Header)
	if err != nil {
		return msg, nil, err
	}
	if !bytes.Equal(msg.Header.Marker[:], bgpMarker) {
		return msg, nil, errors.New("bgp marker invalid")
	}
	bodyReader = io.LimitReader(r, int64(msg.Header.Length)-19)
	return msg, bodyReader, nil
}
