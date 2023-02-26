package bgp

import (
	"FlapAlertedPro/config"
	"encoding/binary"
)

type open struct {
	version          byte // BGP-4
	asn              []byte
	holdTime         uint16
	routerID         uint32
	OptionalParamLen uint8
}

var mpBGP4Cap = []byte{0x01, 0x04, 0x00, 0x01, 0x00, 0x01}
var mpBGP6Cap = []byte{0x01, 0x04, 0x00, 0x02, 0x00, 0x01}
var addPathCap = []byte{0x45, 0x08, 0x00, 0x01, 0x01, 0x01, 0x00, 0x02, 0x01, 0x01}

func fourByteAsnCap() []byte {
	a := []byte{0x41, 0x04}
	return append(a, uint32toByte(uint32(config.GlobalConf.Asn))...)
}

func addHeader(raw []byte, tp msgType) []byte {
	var marker = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	l := uint16(len(raw) + 16 + 2 + 1)
	marker = append(marker, uint16toByte(l)...)
	marker = append(marker, byte(tp))
	marker = append(marker, raw...)
	return marker
}

func getOpen() []byte {
	defaultOpenParameters := open{
		version:  0x4,
		asn:      []byte{0x5b, 0xa0},
		holdTime: 240,
		routerID: 55,
	}
	var r []byte
	if config.GlobalConf.UseAddPath {
		r = constructOpen(defaultOpenParameters, mpBGP4Cap, mpBGP6Cap, fourByteAsnCap(), addPathCap)
	} else {
		r = constructOpen(defaultOpenParameters, mpBGP4Cap, mpBGP6Cap, fourByteAsnCap())
	}
	result := addHeader(r, msgOpen)
	return result
}

func constructOpen(o open, capabilities ...[]byte) []byte {
	result := make([]byte, 0)
	result = append(result, o.version)
	result = append(result, o.asn...)
	result = append(result, uint16toByte(o.holdTime)...)
	result = append(result, uint32toByte(o.routerID)...)

	tempH := make([]byte, 0)
	temp := make([]byte, 0)

	tempH = append(tempH, 0x02) //capability

	for _, c := range capabilities {
		temp = append(temp, c...)
	}
	tLen := uint8(len(temp))
	tempH = append(tempH, tLen) // capabilities length

	tempH = append(tempH, temp...)

	result = append(result, uint8(len(tempH)))
	result = append(result, tempH...)
	return result
}

func uint16toByte(i uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, i)
	return b
}

func uint32toByte(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}
