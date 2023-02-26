package bgp

import (
	"FlapAlertedPro/config"
	"encoding/binary"
	"log"
	"net"
	"strconv"
)

const (
	BA_AS_PATH       = 0x02
	BA_MP_REACH_NLRI = 0x0e

	MPNLRI_AFI_4 = 0x01
	MPNLRI_AFI_6 = 0x02
)

type UserUpdate struct {
	Path      []AsPath
	rawPrefix []rawPrefix
	Prefix    []string
}

type rawPrefix struct {
	Prefix4       []byte
	Prefix6       []byte
	PrefixLenBits int
}

func parseUpdateMsgNew(raw []byte, updateChannel chan *UserUpdate) {
	defer func() {
		if r := recover(); r != nil {
			debugPrintln("Panic at updateParse", r)
		}
	}()

	userUpdate := &UserUpdate{}
	userUpdate.rawPrefix = make([]rawPrefix, 0)
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
	debugPrintf("Attributes: %x\n", attributes)
	parseAttr(attributes, userUpdate)
	pos += int(tPalR)

	nlriInfo := raw[pos:]
	debugPrintf("NLRI: %x\n", nlriInfo)
	debugPrintln("NLRI length should be", len(raw)-int(tPalR)-4)
	parsev4Nlri(nlriInfo, userUpdate)

	pPrefixlist := make([]string, 0)
	for i := range userUpdate.rawPrefix {
		if userUpdate.rawPrefix[i].Prefix4 != nil {
			if len(userUpdate.rawPrefix[i].Prefix4) == 0 {
				continue
			}
			pPrefixlist = append(pPrefixlist, toNetCidr(userUpdate.rawPrefix[i].Prefix4, userUpdate.rawPrefix[i].PrefixLenBits, false))
		}
		if userUpdate.rawPrefix[i].Prefix6 != nil {
			if len(userUpdate.rawPrefix[i].Prefix6) == 0 {
				continue
			}
			pPrefixlist = append(pPrefixlist, toNetCidr(userUpdate.rawPrefix[i].Prefix6, userUpdate.rawPrefix[i].PrefixLenBits, true))
		}
	}

	if len(pPrefixlist) == 0 {
		return
	}
	userUpdate.rawPrefix = nil
	userUpdate.Prefix = pPrefixlist

	debugPrintln("#############################################################################")
	debugPrintln("BGP UPDATE")
	debugPrintln("Prefixes:", userUpdate.Prefix)
	debugPrintln("Paths:", userUpdate.Path)
	debugPrintln("#############################################################################")
	updateChannel <- userUpdate
}

func toNetCidr(prefix []byte, prefixlenBits int, isV6 bool) string {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[WARNING] BGP data format error")
		}
	}()

	if isV6 {
		needBytes := 16 - len(prefix)
		toAppend := make([]byte, needBytes)
		prefix = append(prefix, toAppend...)
		ip := net.IP{prefix[0], prefix[1], prefix[2], prefix[3], prefix[4], prefix[5],
			prefix[6], prefix[7], prefix[8], prefix[9], prefix[10], prefix[11], prefix[12],
			prefix[13], prefix[14], prefix[15]}
		cidr := ip.String() + "/" + strconv.Itoa(prefixlenBits)
		return cidr
	} else {
		needBytes := 4 - len(prefix)
		toAppend := make([]byte, needBytes)
		prefix = append(prefix, toAppend...)
		ip := net.IP{prefix[0], prefix[1], prefix[2], prefix[3]}
		cidr := ip.String() + "/" + strconv.Itoa(prefixlenBits)
		return cidr
	}
}

type AsPath struct {
	Asn []uint32
}

func parseAttr(a []byte, upd *UserUpdate) {
	debugPrintln(":BEGIN ATTRIBUTES:")

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
			attrLen = int(a[pos])
			pos++
		}

		debugPrintln("AttrLen value:", attrLen)
		switch attrType {
		case BA_AS_PATH:
			e := 0
			for e < attrLen {
				debugPrintln("As path attrLen value:", e, attrLen)
				debugPrintf("%x\n", a[pos:pos+attrLen])
				segType := a[pos+e]
				e++
				if segType != 0x02 {
					break
				}

				segLen := int(a[pos+e])
				debugPrintln("Number of ASNs in the path", segLen)
				e++

				newAsPath := AsPath{}
				newAsPath.Asn = make([]uint32, segLen)
				for i := 0; i < segLen; i++ {
					newAsPath.Asn[i] = toUint32([]byte{a[pos+e], a[pos+e+1], a[pos+e+2], a[pos+e+3]})
					e += 4
				}
				upd.Path = append(upd.Path, newAsPath)
				debugPrintln("Found path", newAsPath)
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

			lenNextHop := int(a[pos+e])
			e++

			e = e + lenNextHop // skip next hop
			e++                // skip SNPA
			//BEGIN NLRA
			for e < attrLen {

				if config.GlobalConf.UseAddPath {
					e = e + 4 //skip pathid
				}

				prefixlenBits := int(a[pos+e])
				e++

				actualLen := prefixlenBits
				for actualLen%8 != 0 {
					actualLen++
				}
				actualLen = actualLen / 8
				prefixR := make([]byte, actualLen)
				copy(prefixR, a[pos+e:pos+e+actualLen])
				e = e + actualLen
				prefixObj := rawPrefix{}
				if isV6 {
					prefixObj.Prefix6 = prefixR
				} else {
					prefixObj.Prefix4 = prefixR
				}
				prefixObj.PrefixLenBits = prefixlenBits
				debugPrintf("--> Found new prefix: %x\n", prefixR)
				upd.rawPrefix = append(upd.rawPrefix, prefixObj)
			}

		}
		pos += attrLen

	}
	debugPrintln(":END ATTRS:")

}

func parsev4Nlri(a []byte, upd *UserUpdate) {
	e := 0
	for e < len(a)-1 {
		if config.GlobalConf.UseAddPath {
			e = e + 4 //skip pathId
		}

		prefixLenBits := int(a[e])
		e++

		actualLen := prefixLenBits
		for actualLen%8 != 0 {
			actualLen++
		}
		actualLen = actualLen / 8
		prefixV4 := make([]byte, actualLen)
		copy(prefixV4, a[e:e+actualLen])
		e = e + actualLen
		debugPrintf("--> Found new prefix (v4): %x\n", prefixV4)
		upd.rawPrefix = append(upd.rawPrefix, rawPrefix{Prefix4: prefixV4, PrefixLenBits: prefixLenBits})
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
