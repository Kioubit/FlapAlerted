package bgp

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
)

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

func StartBGP(updates chan *UserUpdate) {

	listener, err := net.Listen("tcp", ":1790")
	if err != nil {
		log.Fatal("[FATAL]", err.Error())
	}
	debugPrintln("Listening on port 1790")
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)
	for {
		conn, err := listener.Accept()
		if err != nil {
			debugPrintln("Error accepting tcp connection", err.Error())
			continue
		}
		debugPrintln("New connection")
		go newBGPConnection(conn, updates)
	}
}

func newBGPConnection(conn net.Conn, updates chan *UserUpdate) {
	var workerCount = 4
	connDetails := &connectionState{}
	connDetails.rawUpdateBytesChan = make(chan *[]byte, 5000)

	var rawChannel = make(chan *[]byte, 25)
	go receiveHeadersWorker(connDetails, rawChannel, conn)

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
			if err != io.EOF {
				panic(err)
			}
		}
		if n == 0 {
			continue
		}

		newBuff := make([]byte, n)
		copy(newBuff, buff[:n])
		debugPrintln("Connection bytes read:", n)
		rawChannel <- &newBuff
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
func receiveHeadersWorker(connDetails *connectionState, ch chan *[]byte, conn net.Conn) {
	for {
		newBuff := <-ch
		if len(*newBuff) == 0 {
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
				log.Println("BGP Connection established with", conn.RemoteAddr().String())
				_, _ = conn.Write(getOpen())
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
				log.Println("BGP Connection lost with", conn.RemoteAddr().String())
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

func readHeaders(raw *[]byte, connDetails *connectionState) []*header {
	defer func() {
		if r := recover(); r != nil {
			debugPrintln("Panic in readHeaders()", r)
		}
	}()

	var marker = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	if connDetails.nextBuffer != nil {
		debugPrintln("nextBuffer is not nil")
		*raw = append(connDetails.nextBuffer, *raw...)
		connDetails.nextBuffer = nil
	}

	headers := make([]*header, 0)
	cov := len(*raw) - 1
	pos := 0
	debugPrintf("%x\n", raw)

	for pos < cov {
		if cov-pos < 19-1 {
			debugPrintln("cov-pos is smaller than 18", cov, pos)
			connDetails.nextBuffer = (*raw)[pos:]
			return headers
		}

		newHeader := &header{}
		for !bytes.Equal((*raw)[pos:pos+16], marker) {
			debugPrintln("CAUTION: Trying to recover from NO BGP")
			pos++
			if pos+16 > cov {
				debugPrintf("---> No BGP recovery", (*raw)[pos:])
				connDetails.nextBuffer = nil
				return headers
			}
		}
		pos += 16

		nextLength := make([]byte, 2)
		nextLength[0] = (*raw)[pos]
		pos++
		nextLength[1] = (*raw)[pos]
		pos++
		l := binary.BigEndian.Uint16(nextLength)
		debugPrintln("Total Length:", l)
		newHeader.length = l

		newHeader.msgType = (*raw)[pos]
		pos++

		realLength := l - 19

		if cov-pos < int(realLength)-1 {
			debugPrintln("smaller than realLength", cov, pos, realLength)
			connDetails.nextBuffer = (*raw)[pos-19:]
			return headers
		}

		newHeader.msg = (*raw)[pos : pos+int(realLength)]
		pos += int(realLength)

		headers = append(headers, newHeader)
	}

	return headers
}
