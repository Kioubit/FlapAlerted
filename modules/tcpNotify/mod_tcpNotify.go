//go:build mod_tcpNotify
// +build mod_tcpNotify

package tcpnotify

import (
	"FlapAlerted/monitor"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

var moduleName = "mod_tcpNotify"

func init() {
	monitor.RegisterModule(&monitor.Module{
		Name:          moduleName,
		Callback:      notify,
		StartComplete: startTcpServer,
	})
}

var (
	connList   []net.Conn
	connListMu sync.Mutex
)

const connListMaxSize = 20

func startTcpServer() {
	addr, err := net.ResolveTCPAddr("tcp", ":8000")
	if err != nil {
		log.Fatal("["+moduleName+"]", err.Error())
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal("["+moduleName+"]", err.Error())
	}
	defer func(listener *net.TCPListener) {
		err := listener.Close()
		if err != nil {
			log.Fatal("["+moduleName+"]", err.Error())
		}
	}(listener)
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}
		_ = conn.SetKeepAlive(true)
		_ = conn.SetKeepAlivePeriod(30 * time.Second)

		connListMu.Lock()
		if len(connList) > connListMaxSize {
			removeFromConnList(0)
		}
		connList = append(connList, conn)
		connListMu.Unlock()
	}
}

func removeFromConnList(i int) {
	if connList[i] != nil {
		_ = connList[i].Close()
	}
	connList[i] = connList[len(connList)-1]
	connList = connList[:len(connList)-1]
}

func notify(f *monitor.Flap) {
	js, err := json.Marshal(&f)
	if err != nil {
		log.Println("[" + moduleName + "] Failed to marshal flap information")
	}
	connListMu.Lock()
	defer connListMu.Unlock()

	for k := 0; k < len(connList); k++ {
		_, err := connList[k].Write(js)
		if err != nil {
			removeFromConnList(k)
			k--
		}
	}
}
