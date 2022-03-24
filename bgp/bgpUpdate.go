package bgp

import (
	"encoding/binary"
)

const (
	BA_AS_PATH       = 0x02
	BA_MP_REACH_NLRI = 0x0e

	MPNLRI_AFI_4 = 0x01
	MPNLRI_AFI_6 = 0x02
)

/*
type update struct {
	withdrawnRoutesLen uint16
	withdrawnRoutes    []byte
	totalPathAttrLen   uint16
	attrs              []byte
	nlri               []byte
}
*/

type UserUpdate struct {
	Path   []AsPath
	Prefix []Prefix
}

type Prefix struct {
	Prefix4       []byte
	Prefix6       []byte
	PrefixLenBits int
}

func parseUpdateMsgNew(raw []byte, updateChannel chan *UserUpdate) {
	userUpdate := &UserUpdate{}
	userUpdate.Prefix = make([]Prefix, 0)
	userUpdate.Path = make([]AsPath, 0)

	pos := 0

	wrL := make([]byte, 2)
	wrL[0] = raw[pos]
	wrL[1] = raw[pos+1]
	pos += 2
	//Withdrawn route length
	wrLR := toUint16(wrL)
	pos += int(wrLR)

	tPAl := make([]byte, 2)
	tPAl[0] = raw[pos]
	tPAl[1] = raw[pos+1]
	pos += 2
	//totalPathAttributes Length
	tPalR := toUint16(tPAl)
	debugPrintln("totalPathAttrLen", tPalR, "vs totalLength", len(raw))
	if tPalR == 0 {
		return
	}

	attributes := raw[pos : pos+int(tPalR)]
	debugPrintf("ATTRS: %x\n", attributes)
	parseAttr(attributes, userUpdate)
	pos += int(tPalR)

	nlriInfo := raw[pos:]
	debugPrintf("NLRI: %x\n", nlriInfo)
	debugPrintln("NLRI length should be", len(raw)-int(tPalR)-4)
	parsev4Nlri(nlriInfo, userUpdate)

	debugPrintln("-----------------------------------------------------------------------------")
	debugPrintln("UPDATE UPDATE UPDATE")
	debugPrintln("Prefixes:", userUpdate.Prefix)
	debugPrintln("Paths:", userUpdate.Path)
	debugPrintln("#############################################################################")
	updateChannel <- userUpdate
}

type AsPath struct {
	Asn []uint32
}

func parseAttr(a []byte, upd *UserUpdate) {
	debugPrintln(":BEGIN ATTRS:")

	pos := 0
	for pos < len(a)-1 {
		attrFlag := a[pos]
		pos++

		extendedLenFlag := false
		if isAttrFlagExtendedLength(attrFlag) {
			debugPrintln("is extended length")
			extendedLenFlag = true
		}

		attrType := a[pos]
		pos++

		var attrLen int
		if extendedLenFlag {
			attrLenR := make([]byte, 2)
			attrLenR[0] = a[pos]
			pos++
			attrLenR[1] = a[pos]
			pos++
			attrLen = int(toUint16(attrLenR))
		} else {
			attrLen = int(uint8(a[pos]))
			pos++
		}

		debugPrintln("ATTRLEN", attrLen)
		switch attrType {
		case BA_AS_PATH:
			e := 0
			for e < attrLen {
				debugPrintln("as=path=pass", e, attrLen)
				debugPrintf("%x\n", a[pos:pos+attrLen])
				segType := a[pos+e]
				e++
				if segType != 0x02 {
					break
				}

				segLen := int(uint8(a[pos+e]))
				debugPrintln("ASes in the path", segLen)
				e++

				newAsPath := AsPath{}
				newAsPath.Asn = make([]uint32, segLen)
				for i := 0; i < segLen; i++ {
					newAsPath.Asn[i] = toUint32([]byte{a[pos+e], a[pos+e+1], a[pos+e+2], a[pos+e+3]})
					e += 4
				}
				upd.Path = append(upd.Path, newAsPath)
				debugPrintln("found path", newAsPath)
			}
		case BA_MP_REACH_NLRI:
			var isV6 = true
			debugPrintf("MPNLRI %x\n", a[pos:pos+attrLen])
			e := 0

			e++ //skip empty byte
			afi := a[pos+e]
			e++

			switch afi {
			case MPNLRI_AFI_4:
				isV6 = false
			case MPNLRI_AFI_6:
			default:
				pos += attrLen
				continue
			}
			e++ //skip SAFI

			lenNextHop := int(uint8(a[pos+e]))
			e++

			e = e + lenNextHop // skip next hop
			e++                // skip SNPA
			//BEGIN NLRA
			for e < attrLen {
				//e = e + 4          //skip pathid

				prefixlenBits := int(uint8(a[pos+e]))
				e++

				actualLen := prefixlenBits
				for actualLen%8 != 0 {
					actualLen++
				}
				actualLen = actualLen / 8
				prefixR := make([]byte, actualLen)
				copy(prefixR, a[pos+e:pos+e+actualLen])
				e = e + actualLen
				prefixObj := Prefix{}
				if isV6 {
					prefixObj.Prefix6 = prefixR
				} else {
					prefixObj.Prefix4 = prefixR
				}
				prefixObj.PrefixLenBits = prefixlenBits
				debugPrintf("--------------------> Found new prefix: %x\n", prefixR)
				upd.Prefix = append(upd.Prefix, prefixObj)
			}

		}
		pos += attrLen

	}
	debugPrintln(":END ATTRS:")

}

func parsev4Nlri(a []byte, upd *UserUpdate) {
	e := 0
	for e < len(a)-1 {
		//e = e + 4          //skip pathid

		prefixlenBits := int(uint8(a[e]))
		e++

		actualLen := prefixlenBits
		for actualLen%8 != 0 {
			actualLen++
		}
		actualLen = actualLen / 8
		prefixv4 := make([]byte, actualLen)
		copy(prefixv4, a[e:e+actualLen])
		e = e + actualLen
		debugPrintf("-v4--v4--v4--v4--v4--v4-> Found new prefix: %x\n", prefixv4)
		upd.Prefix = append(upd.Prefix, Prefix{Prefix4: prefixv4, PrefixLenBits: prefixlenBits})
	}
}

func isAttrFlagExtendedLength(b byte) bool {
	return isBitSet(b, 4)
}

func isBitSet(b byte, pos int) bool {
	return (b & (1 << pos)) != 0
}

func toUint16(s []byte) uint16 {
	return binary.BigEndian.Uint16(s)
}
func toUint32(s []byte) uint32 {
	return binary.BigEndian.Uint32(s)
}
