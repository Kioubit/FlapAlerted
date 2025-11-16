package notification

import (
	"FlapAlerted/bgp/common"
	"bytes"
	"encoding/binary"
)

func (m Msg) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, m.ErrorCode); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, m.ErrorSubCode); err != nil {
		return nil, err
	}
	if _, err := b.Write(m.ErrorData); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func GetNotification(code ErrorCode, subCode ErrorSubCode, data []byte) ([]byte, error) {
	msg := common.BgpMessage{
		Header: common.GetHeader(common.MsgNotification),
		Body: Msg{
			ErrorCode:    code,
			ErrorSubCode: subCode,
			ErrorData:    data,
		},
	}
	return msg.MarshalBinary()
}
