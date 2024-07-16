package update

import (
	"FlapAlerted/bgp/common"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/netip"
)

func ParseMsgUpdate(r io.Reader, defaultAFI AFI, addPathEnabled bool) (msg Msg, err error) {
	msg = Msg{}
	if err := binary.Read(r, binary.BigEndian, &msg.WithdrawnRoutesLength); err != nil {
		return Msg{}, err
	}

	// Reader for withdrawn routes list
	wR := io.LimitReader(r, int64(msg.WithdrawnRoutesLength))
	msg.WithdrawnRoutesList, err = parsePrefixList(wR, defaultAFI, addPathEnabled)

	if err := binary.Read(r, binary.BigEndian, &msg.TotalPathAttributeLength); err != nil {
		return Msg{}, err
	}

	msg.PathAttributes = make([]pathAttribute, 0)

	// Reader for attributes
	aR := io.LimitReader(r, int64(msg.TotalPathAttributeLength))

	for {
		attribute := pathAttribute{}
		attribute.addPathEnabled = addPathEnabled
		if err := binary.Read(aR, binary.BigEndian, &attribute.Flags); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return Msg{}, err
		}
		if err := binary.Read(aR, binary.BigEndian, &attribute.TypeCode); err != nil {
			return Msg{}, err
		}

		var bodyLength uint16 = 0
		if attribute.Flags.isExtendedLength() {
			var length uint16
			if err := binary.Read(aR, binary.BigEndian, &length); err != nil {
				return Msg{}, err
			}
			bodyLength = length
		} else {
			var length uint8
			if err := binary.Read(aR, binary.BigEndian, &length); err != nil {
				return Msg{}, err
			}
			bodyLength = uint16(length)
		}

		attribute.Body = make([]byte, bodyLength)

		if err := binary.Read(aR, binary.BigEndian, &attribute.Body); err != nil {
			return Msg{}, err
		}
		msg.PathAttributes = append(msg.PathAttributes, attribute)
	}

	msg.NetworkLayerReachabilityInformation, err = parsePrefixList(r, defaultAFI, addPathEnabled)
	if err != nil {
		return Msg{}, err
	}
	return msg, nil
}

func parsePrefixList(r io.Reader, afi AFI, addPathEnabled bool) ([]prefix, error) {
	prefixList := make([]prefix, 0)
	for {
		p := prefix{}
		p.AFI = afi

		if addPathEnabled {
			if err := binary.Read(r, binary.BigEndian, &p.PathID); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, err
			}

			if err := binary.Read(r, binary.BigEndian, &p.LengthBits); err != nil {
				return nil, err
			}
		} else {
			if err := binary.Read(r, binary.BigEndian, &p.LengthBits); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, err
			}
		}

		var additive uint8 = 0
		if p.LengthBits%8 != 0 {
			additive += 1
		}
		expectedLength := (p.LengthBits / 8) + additive
		p.Prefix = make([]byte, expectedLength)
		if err := binary.Read(r, binary.BigEndian, &p.Prefix); err != nil {
			return nil, err
		}
		prefixList = append(prefixList, p)
	}
	return prefixList, nil
}

func (f pathAttributeFlags) isOptional() bool {
	return isBitSet(byte(f), 1)
}
func (f pathAttributeFlags) isWellKnown() bool {
	return !isBitSet(byte(f), 1)
}
func (f pathAttributeFlags) isTransitive() bool {
	return !isBitSet(byte(f), 2)
}
func (f pathAttributeFlags) isPartial() bool {
	return !isBitSet(byte(f), 3)
}
func (f pathAttributeFlags) isExtendedLength() bool {
	return isBitSet(byte(f), 4)
}

func parseMultiProtocolUnreachableNLRI(a pathAttribute) (pathAttributeBody, error) {
	if !a.Flags.isWellKnown() {
		return originAttribute{}, errors.New("well-known flag not set")
	}
	r := bytes.NewReader(a.Body)
	result := MPUnReachNLRI{}
	if err := binary.Read(r, binary.BigEndian, &result.AFI); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &result.SAFI); err != nil {
		return nil, err
	}

	if (result.AFI != AFI4 && result.AFI != AFI6) || result.SAFI != UNICAST {
		return nil, errors.New("unknown <AFI,SAFI> combination")
	}

	var err error
	result.Withdrawn, err = parsePrefixList(r, result.AFI, a.addPathEnabled)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func parseMultiProtocolReachableNLRI(a pathAttribute) (pathAttributeBody, error) {
	if !a.Flags.isWellKnown() {
		return nil, errors.New("well-known flag not set")
	}
	r := bytes.NewReader(a.Body)
	result := MPReachNLRI{}
	if err := binary.Read(r, binary.BigEndian, &result.AFI); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &result.SAFI); err != nil {
		return nil, err
	}

	if (result.AFI != AFI4 && result.AFI != AFI6) || result.SAFI != UNICAST {
		return nil, errors.New("unknown <AFI,SAFI> combination")
	}

	if err := binary.Read(r, binary.BigEndian, &result.NextHopLength); err != nil {
		return nil, err
	}

	if result.AFI == AFI4 {
		ip := [4]byte{}
		if err := binary.Read(r, binary.BigEndian, &ip); err != nil {
			return nil, err
		}
		result.NextHop = netip.AddrFrom4(ip)
	} else {
		ip := [16]byte{}
		if err := binary.Read(r, binary.BigEndian, &ip); err != nil {
			return nil, err
		}
		result.NextHop = netip.AddrFrom16(ip)
	}

	if err := binary.Read(r, binary.BigEndian, &result.Reserved); err != nil {
		return nil, err
	}

	var err error
	result.NLRI, err = parsePrefixList(r, result.AFI, a.addPathEnabled)
	if err != nil {
		return nil, fmt.Errorf("error parsing prefixList %w", err)
	}
	return result, nil
}

func parseOriginAttribute(a pathAttribute) (pathAttributeBody, error) {
	if !a.Flags.isWellKnown() {
		return nil, errors.New("well-known flag not set")
	}
	r := bytes.NewReader(a.Body)
	result := originAttribute{}
	if err := binary.Read(r, binary.BigEndian, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func parseAsPathAttribute(a pathAttribute) (pathAttributeBody, error) {
	if !a.Flags.isWellKnown() {
		return nil, errors.New("well-known flag not set")
	}
	r := bytes.NewReader(a.Body)
	result := asPathAttribute{}
	if err := binary.Read(r, binary.BigEndian, &result.PathSegmentType); err != nil {
		if errors.Is(err, io.EOF) {
			// The AS path can be empty in case of iBGP when in the same ASN
			return asPathAttribute{}, nil
		}
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &result.PathSegmentCount); err != nil {
		return nil, err
	}
	result.Value = make([]uint32, result.PathSegmentCount)
	for i := 0; i < int(result.PathSegmentCount); i++ {
		var asn uint32
		if err := binary.Read(r, binary.BigEndian, &asn); err != nil {
			return nil, err
		}
		result.Value[i] = asn
	}
	return result, nil
}

func (a pathAttribute) GetAttribute() (pathAttributeBody, error) {
	switch a.TypeCode {
	case OriginAttr:
		return parseOriginAttribute(a)
	case AsPathAttr:
		return parseAsPathAttribute(a)
	case MultiProtocolReachableNLRIAttr:
		return parseMultiProtocolReachableNLRI(a)
	case MultiProtocolUnreachableNLRIAttr:
		return parseMultiProtocolUnreachableNLRI(a)
	}
	return nil, fmt.Errorf("unknown attribute code %d", a.TypeCode)
}

func isBitSet(b byte, pos int) bool {
	return (b & (1 << pos)) != 0
}

func (u Msg) GetMpReachNLRI() (MPReachNLRI, bool, error) {
	for _, a := range u.PathAttributes {
		if a.TypeCode == MultiProtocolReachableNLRIAttr {
			target, err := a.GetAttribute()
			if err != nil {
				return MPReachNLRI{}, false, err
			}
			return target.(MPReachNLRI), true, nil
		}
	}
	return MPReachNLRI{}, false, nil
}

func (u Msg) GetMpUnReachNLRI() (MPUnReachNLRI, bool, error) {
	for _, a := range u.PathAttributes {
		if a.TypeCode == MultiProtocolUnreachableNLRIAttr {
			target, err := a.GetAttribute()
			if err != nil {
				return MPUnReachNLRI{}, false, err
			}
			return target.(MPUnReachNLRI), true, nil
		}
	}
	return MPUnReachNLRI{}, false, nil
}

func (u Msg) GetAsPaths() ([]common.AsPathList, error) {
	paths := make([]common.AsPathList, 0)
	for _, a := range u.PathAttributes {
		if a.TypeCode == AsPathAttr {
			attribute, err := a.GetAttribute()
			if err != nil {
				return nil, err
			}
			if attribute.(asPathAttribute).PathSegmentType == AsSequence {
				paths = append(paths, common.AsPathList{Asn: attribute.(asPathAttribute).Value})
			}
		}
	}
	return paths, nil
}

func (p prefix) ToNetCidr() netip.Prefix {
	if p.AFI == AFI6 {
		needBytes := 16 - len(p.Prefix)
		if needBytes < 0 {
			return netip.MustParsePrefix("::/0")
		}
		toAppend := make([]byte, needBytes)
		addr := netip.AddrFrom16([16]byte(append(p.Prefix, toAppend...)))
		return netip.PrefixFrom(addr, int(p.LengthBits))
	} else if p.AFI == AFI4 {
		needBytes := 4 - len(p.Prefix)
		if needBytes < 0 {
			return netip.MustParsePrefix("0.0.0.0/0")
		}
		toAppend := make([]byte, needBytes)
		addr := netip.AddrFrom4([4]byte(append(p.Prefix, toAppend...)))
		return netip.PrefixFrom(addr, int(p.LengthBits))
	}
	return netip.MustParsePrefix("::/0")
}
