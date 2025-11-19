package open

import (
	"FlapAlerted/bgp/common"
	"FlapAlerted/bgp/update"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/netip"
)

func (h HoldTime) GetApplicableSeconds() int {
	if h == 0 {
		return 0
	}
	if h < 3 {
		return 3
	}
	return int(h)
}

func ParseMsgOpen(r io.Reader) (msg Msg, err error) {
	if err := binary.Read(r, binary.BigEndian, &msg.Version); err != nil {
		return msg, err
	}
	if msg.Version != 4 {
		return msg, fmt.Errorf("open message version not supported: %d", msg.Version)
	}
	if err := binary.Read(r, binary.BigEndian, &msg.ASN); err != nil {
		return msg, err
	}
	if err := binary.Read(r, binary.BigEndian, &msg.HoldTime); err != nil {
		return msg, err
	}
	if err := binary.Read(r, binary.BigEndian, &msg.RouterID); err != nil {
		return msg, err
	}
	if err := binary.Read(r, binary.BigEndian, &msg.OptionalParameterLength); err != nil {
		return msg, err
	}

	msg.OptionalParameters = make([]OptionalParameter, 0)
	for {
		parameter := OptionalParameter{}
		if err := binary.Read(r, binary.BigEndian, &parameter.ParameterType); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return msg, err
		}
		if err := binary.Read(r, binary.BigEndian, &parameter.ParameterLength); err != nil {
			return msg, err
		}

		pR := io.LimitReader(r, int64(parameter.ParameterLength))
		switch parameter.ParameterType {
		case CapabilityParameter:
			parameter.ParameterValue, err = parseCapabilityParameter(pR)
			if err != nil {
				return msg, err
			}
		default:
			// Skip
			_, err = io.Copy(io.Discard, pR)
			if err != nil {
				return msg, err
			}
		}
		msg.OptionalParameters = append(msg.OptionalParameters, parameter)
	}
	return msg, nil
}

func parseCapabilityParameter(r io.Reader) (result CapabilityList, err error) {
	result.List = make([]CapabilityOptionalParameter, 0)

	for {
		p := CapabilityOptionalParameter{}
		if err := binary.Read(r, binary.BigEndian, &p.CapabilityCode); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return result, err
		}
		if err := binary.Read(r, binary.BigEndian, &p.CapabilityLength); err != nil {
			return result, err
		}

		cr := io.LimitReader(r, int64(p.CapabilityLength))

		switch p.CapabilityCode {
		case CapabilityCodeFourByteASN:
			t := FourByteASNCapability{}
			if err := binary.Read(cr, binary.BigEndian, &t); err != nil {
				return result, err
			}
			p.CapabilityValue = t
		case CapabilityCodeAddPath:
			list := make(AddPathCapabilityList, 0)
			for {
				t := AddPathCapability{}
				if err := binary.Read(cr, binary.BigEndian, &t); err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return result, err
				}
				list = append(list, t)
			}
			p.CapabilityValue = list
		case CapabilityCodeMultiProtocol:
			t := MultiProtocolCapability{}
			if err := binary.Read(cr, binary.BigEndian, &t); err != nil {
				return result, err
			}
			p.CapabilityValue = t
		case CapabilityCodeExtendedMessage:
			t := ExtendedMessageCapability{}
			p.CapabilityValue = t
		case CapabilityCodeHostname:
			t := HostnameCapability{}
			readString := func() (string, error) {
				var length uint8
				if err := binary.Read(r, binary.BigEndian, &length); err != nil {
					return "", err
				}
				buf := make([]byte, length)
				_, err := io.ReadFull(r, buf)
				return string(buf), err
			}
			if t.Hostname, err = readString(); err != nil {
				return result, err
			}
			if t.DomainName, err = readString(); err != nil {
				return result, err
			}
			p.CapabilityValue = t
		case CapabilityCodeExtendedNextHop:
			list := make(ExtendedNextHopCapabilityList, 0)
			for {
				t := ExtendedNextHopCapability{}
				// The SAFI field in this capability is 2 octets in contrast to the regular SAFI field, thus
				// necessitating individual handling of the fields
				if err := binary.Read(cr, binary.BigEndian, &t.AFI); err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return result, err
				}
				safi := [2]byte{}
				if err := binary.Read(cr, binary.BigEndian, &safi); err != nil {
					return result, err
				}
				t.SAFI = update.SAFI(binary.BigEndian.Uint16(safi[:]))
				if err := binary.Read(cr, binary.BigEndian, &t.NextHopAFI); err != nil {
					return result, err
				}
				list = append(list, t)
			}
			p.CapabilityValue = list
		default:
			t := UnknownCapability{}
			t.Value = make([]byte, p.CapabilityLength)
			if err := binary.Read(cr, binary.BigEndian, &t.Value); err != nil {
				return result, err
			}
			p.CapabilityValue = t
		}
		result.List = append(result.List, p)
	}

	return result, nil
}

func GetOpen(holdTime int, routerID netip.Addr, capabilities ...CapabilityOptionalParameter) ([]byte, error) {
	routerIDBytes := routerID.As4()
	routerIDUint32 := binary.BigEndian.Uint32(routerIDBytes[:])

	msg := common.BgpMessage{
		Header: common.GetHeader(common.MsgOpen),
		Body: Msg{
			Version:                 4,
			ASN:                     ASTrans,
			HoldTime:                HoldTime(holdTime),
			RouterID:                RouterID(routerIDUint32),
			OptionalParameterLength: 0,
			OptionalParameters: []OptionalParameter{
				{
					ParameterType:   CapabilityParameter,
					ParameterLength: 0,
					ParameterValue:  CapabilityList{List: capabilities},
				},
			},
		},
	}

	return msg.MarshalBinary()
}
