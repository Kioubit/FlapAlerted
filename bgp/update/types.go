package update

import (
	"FlapAlerted/bgp/common"
	"net/netip"
)

type Msg struct {
	WithdrawnRoutesLength               uint16
	WithdrawnRoutesList                 []prefix
	TotalPathAttributeLength            uint16
	PathAttributes                      []pathAttribute
	NetworkLayerReachabilityInformation []prefix
}

type prefix struct {
	AFI        common.AFI // Added
	PathID     uint32
	LengthBits uint8
	Prefix     []byte
}

type pathAttribute struct {
	Flags    pathAttributeFlags
	TypeCode pathAttributeType
	Body     []byte
}

type pathAttributeFlags byte

type pathAttributeBody any

type pathAttributeType uint8

const (
	OriginAttr                       pathAttributeType = 1
	AsPathAttr                       pathAttributeType = 2
	MultiProtocolReachableNLRIAttr   pathAttributeType = 14
	MultiProtocolUnreachableNLRIAttr pathAttributeType = 15
)

type pathSegmentType uint8

const (
	AsSet      pathSegmentType = 1
	AsSequence pathSegmentType = 2
)

type MPReachNLRI struct {
	AFI           common.AFI
	SAFI          common.SAFI
	NextHopLength uint8
	NextHop       []netip.Addr
	Reserved      uint8
	NLRI          []prefix
}

type MPUnReachNLRI struct {
	AFI       common.AFI
	SAFI      common.SAFI
	Withdrawn []prefix
}

type OriginType uint8

const (
	originIGP     OriginType = 0
	originEGP     OriginType = 1
	originUnknown OriginType = 2
)

type originAttribute struct {
	Origin OriginType
}

type asPathAttribute struct {
	Segments []asPathAttributeSegment
}

type asPathAttributeSegment struct {
	PathSegmentType  pathSegmentType
	PathSegmentCount uint8
	Value            common.AsPath
}
