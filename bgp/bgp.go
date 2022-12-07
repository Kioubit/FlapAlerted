package bgp

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

var GlobalAddPath = false

type msgType byte

const (
	msgOpen         msgType = 0x01
	msgUpdate       msgType = 0x02
	msgNotification msgType = 0x03
	msgKeepAlive    msgType = 0x04
)

type header struct {
	marker  [16]byte
	length  uint16
	msgType byte
	msg     []byte
}

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

func fourByteAsnCap(asn uint32) []byte {
	a := []byte{0x41, 0x04}
	a = append(a, uint32toByte(asn)...)
	return a
}

func StartBGP(asn uint32, updates chan *UserUpdate) {
	listener, err := net.Listen("tcp", ":1790")
	if err != nil {
		log.Fatal("[FATAL]", err.Error())
	}
	debugPrintln("Listening")
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			debugPrintln("Error accepting tcp connection", err.Error())
			continue
		}
		debugPrintln("New connection")
		go newBGPConnection(conn, asn, updates)
	}
}

func rawUpdateMessageWorker(channel chan *[]byte, user chan *UserUpdate) {
	for {
		u := <-channel
		if u == nil {
			break
		}
		parseUpdateMsgNew(*u, user)
	}
}

func newBGPConnection(conn net.Conn, asn uint32, updates chan *UserUpdate) {
	var workerCount = 4
	connDetails := &connectionState{}
	connDetails.rawUpdateBytesChan = make(chan *[]byte, 5000)

	var rawChannel = make(chan []byte, 25)
	go receiveHeadersWorker(connDetails, rawChannel, asn, conn)

	defer func() {
		if r := recover(); r != nil {
			debugPrintln("Panic", r)
			if conn != nil {
				_ = conn.Close()
				close(rawChannel)
				close(connDetails.rawUpdateBytesChan)
			}
		}
	}()

	for i := 0; i < workerCount; i++ {
		go rawUpdateMessageWorker(connDetails.rawUpdateBytesChan, updates)
	}

	const BGPBuffSize = 10000 * 1000
	buff := make([]byte, BGPBuffSize) //Reused
	for {
		n, err := conn.Read(buff)
		if err != nil {
			panic(err)
		}
		if n == 0 {
			continue
		}

		newBuff := make([]byte, n)
		copy(newBuff, buff[:n])
		debugPrintln("READ", n)
		rawChannel <- newBuff
	}
}

func receiveHeadersWorker(connDetails *connectionState, ch chan []byte, asn uint32, conn net.Conn) {
	for {
		newBuff := <-ch
		if len(newBuff) == 0 {
			return
		}
		headers := readHeaders(newBuff, connDetails)
		if headers == nil {
			continue
		}
		for i := range headers {
			switch headers[i].msgType {
			case byte(msgOpen):
				debugPrintln("Received BGP OPEN Message. Replying with OPEN")
				_, _ = conn.Write(getOpen(asn))
			case byte(msgKeepAlive):
				debugPrintln("Received BGP KEEPALIVE Message")
				_, _ = conn.Write(addHeader(make([]byte, 0), msgKeepAlive))
			case byte(msgUpdate):
				debugPrintln("Received BGP UPDATE MESSAGE", len(headers[i].msg))
				debugPrintf("%x\n", headers[i].msg)
				connDetails.rawUpdateBytesChan <- &headers[i].msg
			default:
				debugPrintln("Received unknown BGP Message. Closing connection")
				debugPrintln("BGP Error notification")
				_ = conn.Close()
				return
			}
		}
	}

}

type connectionState struct {
	nextBuffer         []byte
	rawUpdateBytesChan chan *[]byte
}

func readHeaders(raw []byte, connDetails *connectionState) []*header {
	defer func() {
		if r := recover(); r != nil {
			debugPrintln("Panic in readHeaders()", r)
		}
	}()

	var marker = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	if connDetails.nextBuffer != nil {
		debugPrintln("nextBuffer is not nil")
		raw = append(connDetails.nextBuffer, raw...)
		connDetails.nextBuffer = nil
	}

	headers := make([]*header, 0)
	cov := len(raw) - 1
	pos := 0
	debugPrintf("%x\n", raw)

	for pos < cov {
		if cov-pos < 19-1 {
			debugPrintln("cov-pos is smaller than 18", cov, pos)
			connDetails.nextBuffer = raw[pos:]
			return headers
		}

		newHeader := &header{}
		for !bytes.Equal(raw[pos:pos+16], marker) {
			debugPrintln("CAUTION: Trying to recover from NO BGP")
			pos++
			if pos+16 > cov {
				debugPrintf("---> No BGP recovery", raw[pos:])
				connDetails.nextBuffer = nil
				return headers
			}
		}
		pos += 16

		nextLength := make([]byte, 2)
		nextLength[0] = raw[pos]
		pos++
		nextLength[1] = raw[pos]
		pos++
		l := binary.BigEndian.Uint16(nextLength)
		debugPrintln("Total Length:", l)
		newHeader.length = l

		newHeader.msgType = raw[pos]
		pos++

		realLength := l - 19

		if cov-pos < int(realLength)-1 {
			debugPrintln("smaller than realLength", cov, pos, realLength)
			connDetails.nextBuffer = raw[pos-19:]
			return headers
		}

		newHeader.msg = raw[pos : pos+int(realLength)]
		pos += int(realLength)

		headers = append(headers, newHeader)
	}

	return headers
}

func getOpen(asn uint32) []byte {
	defaultOpenParameters := open{
		version:  0x4,
		asn:      []byte{0x5b, 0xa0},
		holdTime: 240,
		routerID: 55,
	}
	var r []byte
	if GlobalAddPath {
		r = constructOpen(defaultOpenParameters, mpBGP4Cap, mpBGP6Cap, fourByteAsnCap(asn), addPathCap)
	} else {
		r = constructOpen(defaultOpenParameters, mpBGP4Cap, mpBGP6Cap, fourByteAsnCap(asn))
	}
	result := addHeader(r, msgOpen)
	return result
}

func addHeader(raw []byte, tp msgType) []byte {
	var marker = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	l := uint16(len(raw) + 16 + 2 + 1)
	marker = append(marker, uint16toByte(l)...)
	marker = append(marker, byte(tp))
	marker = append(marker, raw...)
	return marker
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
