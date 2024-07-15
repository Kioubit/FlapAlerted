package bgp

import "FlapAlerted/bgp/common"

type keepAliveMsg struct {
}

func (m keepAliveMsg) MarshalBinary() (data []byte, err error) {
	return []byte{}, nil
}

func GetKeepAlive() ([]byte, error) {
	m := common.BgpMessage{
		Header: common.GetHeader(common.MsgKeepAlive),
		Body:   keepAliveMsg{},
	}
	return m.MarshalBinary()
}
