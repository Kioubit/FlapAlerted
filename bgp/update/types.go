package update

import (
	"encoding/json"
	"log/slog"
	"net/netip"
)

type Msg struct {
	WithdrawnRoutesLength               uint16
	WithdrawnRoutesList                 []prefix
	TotalPathAttributeLength            uint16
	PathAttributes                      []pathAttribute
	NetworkLayerReachabilityInformation []prefix
}

func (u Msg) LogValue() slog.Value {
	k, err := json.Marshal(u)
	if err != nil {
		return slog.StringValue("Failed to marshal update to JSON: " + err.Error())
	}

	asPath, err := u.GetAsPaths()
	if err != nil {
		slog.Warn("error getting ASPath", "error", err)
	}

	nlRi, foundNlRi, err := u.GetMpReachNLRI()
	if err != nil {
		slog.Warn("error getting MpReachNLRI", "error", err)
	}
	prefixList := make([]string, 0)
	if foundNlRi {
		for i := range nlRi.NLRI {
			prefixList = append(prefixList, nlRi.NLRI[i].ToNetCidr().String())
		}
	}

	for _, information := range u.NetworkLayerReachabilityInformation {
		prefixList = append(prefixList, information.ToNetCidr().String())
	}

	return slog.GroupValue(
		slog.Attr{
			Key:   "full",
			Value: slog.StringValue(string(k)),
		},
		slog.Attr{
			Key:   "asPath",
			Value: slog.AnyValue(asPath),
		},
		slog.Attr{
			Key:   "prefixes",
			Value: slog.AnyValue(prefixList),
		},
	)
}

type prefix struct {
	AFI        AFI // Added
	PathID     uint32
	LengthBits uint8
	Prefix     []byte
}

type pathAttribute struct {
	Flags                pathAttributeFlags
	TypeCode             pathAttributeType
	Body                 []byte
	addPathEnabled       bool // Added
	hasExtendedNextHopV4 bool //Added
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

type AFI uint16

const (
	AFI4 AFI = 1
	AFI6 AFI = 2
)

type SAFI uint8

const (
	UNICAST   SAFI = 1
	MULTICAST SAFI = 2
)

type MPReachNLRI struct {
	AFI           AFI
	SAFI          SAFI
	NextHopLength uint8
	NextHop       []netip.Addr
	Reserved      uint8
	NLRI          []prefix
}

type MPUnReachNLRI struct {
	AFI       AFI
	SAFI      SAFI
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
	PathSegmentType  pathSegmentType
	PathSegmentCount uint8
	Value            []uint32
}
