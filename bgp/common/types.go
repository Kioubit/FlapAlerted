package common

import "slices"

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

type AsPath []uint32

func (p AsPath) Equals(other AsPath) bool {
	return slices.Equal(p, other)
}
