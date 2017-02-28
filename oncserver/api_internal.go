package oncserver

import (
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/swiftstack/onc"
)

type serverProgVersSetMap map[uint32]bool // Program Version Set (map entry exists and is true if the program supports the map key's version)

type serverProgSetMap map[uint32]serverProgVersSetMap // Program Set (map entry exists if the map key's program is supported)

type tcpServerConnStruct struct {
	sync.Mutex
	sync.WaitGroup
	tcpConn *net.TCPConn
}

type tcpServerStruct struct {
	sync.Mutex
	sync.WaitGroup
	port             uint16
	serverProgSet    serverProgSetMap
	tcpAddr          *net.TCPAddr
	tcpListener      *net.TCPListener
	tcpServerConnMap map[*net.TCPConn]*tcpServerConnStruct
}

type udpServerStruct struct {
	sync.Mutex
	sync.WaitGroup
	port          uint16
	serverProgSet serverProgSetMap
	udpAddr       *net.UDPAddr
	udpConn       *net.UDPConn
}

type globalsStruct struct {
	sync.Mutex
	tcpServerMap map[uint16]*tcpServerStruct // Key is tcpServerStruct.port
	udpServerMap map[uint16]*udpServerStruct // Key is udpServerStruct.port
}

var globals globalsStruct

func init() {
	globals.tcpServerMap = make(map[uint16]*tcpServerStruct)
	globals.udpServerMap = make(map[uint16]*udpServerStruct)
}

func startServer(prot uint32, port uint16, progVersList []ProgVersStruct, callbacks RequestCallbacks) (err error) {
	var (
		ok                bool
		progVers          ProgVersStruct
		serverProgSet     serverProgSetMap
		serverProgVersSet serverProgVersSetMap
		tcpServer         *tcpServerStruct
		udpServer         *udpServerStruct
		vers              uint32
	)

	serverProgSet = make(map[uint32]serverProgVersSetMap)

	for _, progVers = range progVersList {
		_, ok = serverProgSet[progVers.Prog]
		if ok {
			err = fmt.Errorf("prog (%v) appears more than once in progVersList", progVers.Prog)
			return
		}
		serverProgVersSet = make(map[uint32]bool)
		serverProgSet[progVers.Prog] = serverProgVersSet
		for _, vers = range progVers.VersList {
			_, ok = serverProgVersSet[vers]
			if ok {
				err = fmt.Errorf("vers (%v) appears more than once for prog (%v) in progVersList", vers, progVers.Prog)
				return
			}
			serverProgVersSet[vers] = true
		}
	}

	switch prot {
	case onc.IPProtoTCP:
		tcpServer = &tcpServerStruct{
			port:             port,
			serverProgSet:    serverProgSet,
			tcpServerConnMap: make(map[*net.TCPConn]*tcpServerConnStruct),
		}

		tcpServer.tcpAddr, err = net.ResolveTCPAddr("tcp4", ":"+strconv.FormatUint(uint64(tcpServer.port), 10))
		if nil != err {
			return
		}

		tcpServer.tcpListener, err = net.ListenTCP("tcp4", tcpServer.tcpAddr)
		if nil != err {
			return
		}

		globals.Lock()

		_, ok = globals.tcpServerMap[tcpServer.port]
		if ok {
			globals.Unlock()
			err = fmt.Errorf("logic error: globals.tcpServerMap already contains specified port (%v)", tcpServer.port)
			panic(err)
		}

		globals.tcpServerMap[tcpServer.port] = tcpServer

		tcpServer.WaitGroup.Add(1)

		go tcpServer.tcpListenerHandler()

		globals.Unlock()
	case onc.IPProtoUDP:
		udpServer = &udpServerStruct{
			port:          port,
			serverProgSet: serverProgSet,
		}

		udpServer.udpAddr, err = net.ResolveUDPAddr("udp4", ":"+strconv.FormatUint(uint64(udpServer.port), 10))
		if nil != err {
			return
		}

		udpServer.udpConn, err = net.ListenUDP("udp4", udpServer.udpAddr)
		if nil != err {
			return
		}

		globals.Lock()

		_, ok = globals.udpServerMap[udpServer.port]
		if ok {
			globals.Unlock()
			err = fmt.Errorf("logic error: globals.udpServerMap already contains specified port (%v)", udpServer.port)
			panic(err)
		}

		globals.udpServerMap[udpServer.port] = udpServer

		udpServer.WaitGroup.Add(1)

		go udpServer.udpConnHandler()

		globals.Unlock()
	default:
		err = fmt.Errorf("invalid prot (%v) - must be either onc.IPProtoTCP (%v) or onc.IPProtoUDP (%v)", prot, onc.IPProtoTCP, onc.IPProtoUDP)
		return
	}

	err = nil
	return
}

func stopServer(prot uint32, port uint16) (err error) {
	var (
		ok            bool
		tcpServer     *tcpServerStruct
		tcpServerConn *tcpServerConnStruct
		udpServer     *udpServerStruct
	)

	switch prot {
	case onc.IPProtoTCP:
		globals.Lock()
		tcpServer, ok = globals.tcpServerMap[port]
		if !ok {
			globals.Unlock()
			err = fmt.Errorf("no tcpServer registered for port %v", port)
			return
		}
		delete(globals.tcpServerMap, port)
		globals.Unlock()
		tcpServer.Lock()
		err = tcpServer.tcpListener.Close()
		if nil != err {
			tcpServer.Unlock()
			return
		}
		for _, tcpServerConn = range tcpServer.tcpServerConnMap {
			tcpServerConn.Lock()
			err = tcpServerConn.tcpConn.Close()
			if nil != err {
				tcpServerConn.Unlock()
				tcpServer.Unlock()
				return
			}
			tcpServerConn.Unlock()
			tcpServerConn.Wait()
		}
		tcpServer.Wait()
	case onc.IPProtoUDP:
		globals.Lock()
		udpServer, ok = globals.udpServerMap[port]
		if !ok {
			globals.Unlock()
			err = fmt.Errorf("no udpServer registered for port %v", port)
			return
		}
		delete(globals.udpServerMap, port)
		globals.Unlock()
		udpServer.Lock()
		err = udpServer.udpConn.Close()
		if nil != err {
			udpServer.Unlock()
			return
		}
		udpServer.Unlock()
		if nil != err {
			return
		}
		udpServer.Wait()
	default:
		err = fmt.Errorf("invalid prot (%v) - must be either onc.IPProtoTCP (%v) or onc.IPProtoUDP (%v)", prot, onc.IPProtoTCP, onc.IPProtoUDP)
		return
	}

	err = nil
	return
}

func sendDeniedRpcMismatchReply(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, mismatchInfoLow uint32, mismatchInfoHigh uint32) (err error) {
	err = nil // TODO
	return
}

func sendDeniedAuthErrorReply(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, stat uint32) (err error) {
	err = nil // TODO
	return
}

func sendAcceptedSuccess(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, results []byte) (err error) {
	err = nil // TODO
	return
}

func sendAcceptedProgMismatchReply(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, mismatchInfoLow uint32, mismatchInfoHigh uint32) (err error) {
	err = nil // TODO
	return
}

func sendAcceptedOtherErrorReply(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, stat uint32) (err error) {
	err = nil // TODO
	return
}
