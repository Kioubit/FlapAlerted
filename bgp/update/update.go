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

func ParseMsgUpdate(r io.Reader, defaultAFI common.AFI, addPathEnabled bool) (msg Msg, err error) {
	msg = Msg{}
	if err := binary.Read(r, binary.BigEndian, &msg.WithdrawnRoutesLength); err != nil {
		return Msg{}, err
	}

	// Reader for withdrawn routes list
	wR := io.LimitReader(r, int64(msg.WithdrawnRoutesLength))
	msg.WithdrawnRoutesList, err = parsePrefixList(wR, defaultAFI, addPathEnabled)
	if err != nil {
		return Msg{}, err
	}

	if err := binary.Read(r, binary.BigEndian, &msg.TotalPathAttributeLength); err != nil {
		return Msg{}, err
	}

	msg.PathAttributes = make([]pathAttribute, 0)

	// Reader for attributes
	aR := io.LimitReader(r, int64(msg.TotalPathAttributeLength))

	for {
		attribute := pathAttribute{}
		if err := binary.Read(aR, binary.BigEndian, &attribute.Flags); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return Msg{}, err
		}
		if err := binary.Read(aR, binary.BigEndian, &attribute.TypeCode); err != nil {
			return Msg{}, err
		}

		var bodyLength uint16
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

		if _, err := io.ReadFull(aR, attribute.Body); err != nil {
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

func parsePrefixList(r io.Reader, afi common.AFI, addPathEnabled bool) ([]prefix, error) {
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
		if _, err := io.ReadFull(r, p.Prefix); err != nil {
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

func parseMultiProtocolUnreachableNLRI(a pathAttribute, session *common.LocalSession) (pathAttributeBody, error) {
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

	if (result.AFI != common.AFI4 && result.AFI != common.AFI6) || result.SAFI != common.UNICAST {
		return nil, errors.New("unknown <AFI,SAFI> combination")
	}

	var err error
	result.Withdrawn, err = parsePrefixList(r, result.AFI, session.AddPathEnabled)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func parseMultiProtocolReachableNLRI(a pathAttribute, session *common.LocalSession) (pathAttributeBody, error) {
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

	if (result.AFI != common.AFI4 && result.AFI != common.AFI6) || result.SAFI != common.UNICAST {
		return nil, errors.New("unknown <AFI,SAFI> combination")
	}

	if err := binary.Read(r, binary.BigEndian, &result.NextHopLength); err != nil {
		return nil, err
	}

	afiReader := io.LimitReader(r, int64(result.NextHopLength))

	if result.NextHopLength == 0 {
		result.NextHop = make([]netip.Addr, 0)
	}

	// Extended next hops
	nextHopAfi := result.AFI
	if session.HasExtendedNextHopV4 && nextHopAfi == common.AFI4 {
		if result.NextHopLength == 16 || result.NextHopLength == 32 {
			nextHopAfi = common.AFI6
		}
	}

	if nextHopAfi == common.AFI4 {
		result.NextHop = make([]netip.Addr, 0, result.NextHopLength/4)
		ip := [4]byte{}
		for {
			if _, err := io.ReadFull(afiReader, ip[:]); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, err
			}
			result.NextHop = append(result.NextHop, netip.AddrFrom4(ip))
		}
	} else {
		result.NextHop = make([]netip.Addr, 0, result.NextHopLength/16)
		ip := [16]byte{}
		for {
			if _, err := io.ReadFull(afiReader, ip[:]); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, err
			}
			result.NextHop = append(result.NextHop, netip.AddrFrom16(ip))
		}
	}

	if err := binary.Read(r, binary.BigEndian, &result.Reserved); err != nil {
		return nil, err
	}

	var err error
	result.NLRI, err = parsePrefixList(r, result.AFI, session.AddPathEnabled)
	if err != nil {
		return nil, fmt.Errorf("error parsing prefixList: %w", err)
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
	result := asPathAttribute{
		Segments: make([]asPathAttributeSegment, 0, 1),
	}
	for {
		newSegment := asPathAttributeSegment{}
		if err := binary.Read(r, binary.BigEndian, &newSegment.PathSegmentType); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if err := binary.Read(r, binary.BigEndian, &newSegment.PathSegmentCount); err != nil {
			return nil, err
		}
		newSegment.Value = make([]uint32, newSegment.PathSegmentCount)
		for i := 0; i < int(newSegment.PathSegmentCount); i++ {
			var asn uint32
			if err := binary.Read(r, binary.BigEndian, &asn); err != nil {
				return nil, err
			}
			newSegment.Value[i] = asn
		}
		result.Segments = append(result.Segments, newSegment)
	}

	return result, nil
}

func (a pathAttribute) GetAttribute(session *common.LocalSession) (pathAttributeBody, error) {
	switch a.TypeCode {
	case OriginAttr:
		return parseOriginAttribute(a)
	case AsPathAttr:
		return parseAsPathAttribute(a)
	case MultiProtocolReachableNLRIAttr:
		return parseMultiProtocolReachableNLRI(a, session)
	case MultiProtocolUnreachableNLRIAttr:
		return parseMultiProtocolUnreachableNLRI(a, session)
	}
	return nil, fmt.Errorf("unknown attribute code %d", a.TypeCode)
}

func isBitSet(b byte, pos int) bool {
	return (b & (1 << pos)) != 0
}

func (u Msg) GetMpReachNLRI(session *common.LocalSession) (MPReachNLRI, bool, error) {
	for _, a := range u.PathAttributes {
		if a.TypeCode == MultiProtocolReachableNLRIAttr {
			target, err := a.GetAttribute(session)
			if err != nil {
				return MPReachNLRI{}, false, err
			}
			return target.(MPReachNLRI), true, nil
		}
	}
	return MPReachNLRI{}, false, nil
}

func (u Msg) GetMpUnReachNLRI(session *common.LocalSession) (MPUnReachNLRI, bool, error) {
	for _, a := range u.PathAttributes {
		if a.TypeCode == MultiProtocolUnreachableNLRIAttr {
			target, err := a.GetAttribute(session)
			if err != nil {
				return MPUnReachNLRI{}, false, err
			}
			return target.(MPUnReachNLRI), true, nil
		}
	}
	return MPUnReachNLRI{}, false, nil
}

func (u Msg) GetAsPath(session *common.LocalSession) (common.AsPath, bool, error) {
	path := make(common.AsPath, 0)
	found := false

OuterLoop:
	for _, a := range u.PathAttributes {
		switch a.TypeCode {
		case AsPathAttr:
			found = true
			attribute, err := a.GetAttribute(session)
			if err != nil {
				return nil, false, err
			}
			for _, segment := range attribute.(asPathAttribute).Segments {
				if segment.PathSegmentType == AsSequence {
					path = append(path, segment.Value...)
				}
			}
			// A path attribute of a given type can only appear once
			break OuterLoop
		}
	}

	if !found {
		return nil, false, nil
	}

	if len(path) == 0 {
		// The AS path can be completely empty in case of iBGP when in the same ASN
		path = append(path, session.Asn)
	}
	return path, true, nil
}

func (p prefix) ToNetCidr() netip.Prefix {
	switch p.AFI {
	case common.AFI6:
		if len(p.Prefix) > 16 {
			return netip.MustParsePrefix("::/0")
		}
		var addrBytes [16]byte
		copy(addrBytes[:], p.Prefix)
		addr := netip.AddrFrom16(addrBytes)
		return netip.PrefixFrom(addr, int(p.LengthBits))
	case common.AFI4:
		if len(p.Prefix) > 4 {
			return netip.MustParsePrefix("0.0.0.0/0")
		}
		var addrBytes [4]byte
		copy(addrBytes[:], p.Prefix)
		addr := netip.AddrFrom4(addrBytes)
		return netip.PrefixFrom(addr, int(p.LengthBits))
	}
	return netip.MustParsePrefix("::/0")
}
