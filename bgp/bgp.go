package bgp

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

var GlobalAdpath = false

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
	a = append(a, uint32tobyte(asn)...)
	return a
}

func StartBGP(asn uint32, updates chan *UserUpdate) {
	listener, err := net.Listen("tcp", ":1790")
	if err != nil {
		log.Fatal(err)
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

func rawUpdateMesageWorker(channel chan *[]byte, user chan *UserUpdate) {
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

	defer func() {
		if r := recover(); r != nil {
			debugPrintln("Panic", r)
			if conn != nil {
				conn.Close()
				close(connDetails.rawUpdateBytesChan)
			}
		}
	}()

	for i := 0; i < workerCount; i++ {
		go rawUpdateMesageWorker(connDetails.rawUpdateBytesChan, updates)
	}

	const BGPBuffSize = 10000 * 1000
	buff := make([]byte, BGPBuffSize) //Reused
	for {
		n, err := conn.Read(buff)
		if err != nil {
			panic(err)
		}

		newBuff := make([]byte, n)
		copy(newBuff, buff[:n])
		debugPrintln("READ", len(newBuff), n)

		headers := readHeaders(newBuff, connDetails)
		for i := range headers {
			switch headers[i].msgType {
			case byte(msgOpen):
				debugPrintln("Received BGP OPEN Message. Replying with OPEN")
				conn.Write(getOpen(asn))
			case byte(msgKeepAlive):
				debugPrintln("Received BGP KEEPALIVE Message")
				conn.Write(addHeader(make([]byte, 0), msgKeepAlive))
			case byte(msgUpdate):
				debugPrintln("received BGP UPDATE MESSAGE", len(headers[i].msg))
				debugPrintf("%x\n", headers[i].msg)
				connDetails.rawUpdateBytesChan <- &headers[i].msg
			default:
				debugPrintln("Received BGP UNKNOWN Message. Closing connection")
				debugPrintln("BGP Error notification")
				close(connDetails.rawUpdateBytesChan)
				conn.Close()
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
	var marker = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	//for next
	if connDetails.nextBuffer != nil {
		debugPrintln("nextBuf not nil")
		raw = append(connDetails.nextBuffer, raw...)
		connDetails.nextBuffer = nil
	}

	headers := make([]*header, 0)
	cov := len(raw) - 1
	pos := 0
	debugPrintf("%x\n", raw)

	for pos < cov {
		//for next
		if cov-pos < 19-1 {
			debugPrintln("smaller than 19", cov, pos)
			connDetails.nextBuffer = raw[pos:]
			return headers
		}

		newHeader := &header{}
		if !bytes.Equal(raw[pos:pos+16], marker) {
			panic("NO BGP")
		}
		pos += 16

		nextLenght := make([]byte, 2)
		nextLenght[0] = raw[pos]
		pos++
		nextLenght[1] = raw[pos]
		pos++
		l := binary.BigEndian.Uint16(nextLenght)
		debugPrintln("Total Length:", l)
		newHeader.length = l

		newHeader.msgType = raw[pos]
		pos++

		realLength := l - 19

		//for next
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

	defaultOpenParamters := open{
		version:  0x4,
		asn:      []byte{0x5b, 0xa0},
		holdTime: 240,
		routerID: 55,
	}
	var r []byte
	if GlobalAdpath {
		r = constructOpen(defaultOpenParamters, mpBGP4Cap, mpBGP6Cap, fourByteAsnCap(asn), addPathCap)
	} else {
		r = constructOpen(defaultOpenParamters, mpBGP4Cap, mpBGP6Cap, fourByteAsnCap(asn))
	}
	result := addHeader(r, msgOpen)
	return result
}

func addHeader(raw []byte, tp msgType) []byte {
	var marker = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	l := uint16(len(raw) + 16 + 2 + 1)
	marker = append(marker, uint16tobyte(l)...)
	marker = append(marker, byte(tp))
	marker = append(marker, raw...)
	return marker
}

func constructOpen(o open, capabilities ...[]byte) []byte {
	result := make([]byte, 0)
	result = append(result, o.version)
	result = append(result, o.asn...)
	result = append(result, uint16tobyte(o.holdTime)...)
	result = append(result, uint32tobyte(o.routerID)...)

	tempH := make([]byte, 0)
	temp := make([]byte, 0)

	tempH = append(tempH, 0x02) //capability

	for _, c := range capabilities {
		temp = append(temp, c...)
	}
	tlen := uint8(len(temp))
	tempH = append(tempH, tlen) // capabilities length

	tempH = append(tempH, temp...)

	result = append(result, uint8(len(tempH)))
	result = append(result, tempH...)
	return result
}

func uint16tobyte(i uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, i)
	return b
}

func uint32tobyte(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}
