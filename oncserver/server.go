package oncserver

import (
	"net"
	"sync"
)

const (
	maxUdpPacketSize = uint32(65536) // Typically far less than this (e.g. ~2KB or ~7.5KB)
)

type serverProgVersSetMap map[uint32]bool // Program Version Set (map entry exists and is true if the program supports the map key's version)

type serverProgSetMap map[uint32]serverProgVersSetMap // Program Set (map entry exists if the map key's program is supported)

type tcpServerConnStruct struct {
	sync.Mutex
	sync.WaitGroup
	tcpConn   *net.TCPConn
	tcpServer *tcpServerStruct
}

type tcpServerStruct struct {
	sync.Mutex
	sync.WaitGroup
	port             uint16
	serverProgSet    serverProgSetMap
	callbacks        RequestCallbacks
	tcpAddr          *net.TCPAddr
	tcpListener      *net.TCPListener
	tcpServerConnMap map[*net.TCPConn]*tcpServerConnStruct
}

type udpServerConnStruct struct {
	udpAddr   *net.UDPAddr // UDP is actually connectionless, so we just remember remote UDPAddr here
	udpServer *udpServerStruct
}

type udpServerStruct struct {
	sync.Mutex
	sync.WaitGroup
	port          uint16
	serverProgSet serverProgSetMap
	callbacks     RequestCallbacks
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

func (tcpServer *tcpServerStruct) tcpListenerHandler() {
	var (
		err           error
		tcpConn       *net.TCPConn
		tcpServerConn *tcpServerConnStruct
	)

	for {
		tcpConn, err = tcpServer.tcpListener.AcceptTCP()
		if nil != err {
			tcpServer.WaitGroup.Done()
			return
		}

		tcpServerConn = &tcpServerConnStruct{
			tcpConn:   tcpConn,
			tcpServer: tcpServer,
		}

		if nil == tcpServerConn {
		} // TODO
	}
}

func (udpServer *udpServerStruct) udpConnHandler() {
	var (
		err           error
		udpAddr       *net.UDPAddr
		udpPacketBuf  []byte
		udpPacketLen  int
		udpServerConn *udpServerConnStruct
	)

	for {
		udpPacketBuf = make([]byte, maxUdpPacketSize, maxUdpPacketSize)

		udpPacketLen, udpAddr, err = udpServer.udpConn.ReadFromUDP(udpPacketBuf)
		if nil != err {
			udpServer.WaitGroup.Done()
			return
		}

		udpPacketBuf = udpPacketBuf[:udpPacketLen]

		udpServerConn = &udpServerConnStruct{
			udpAddr:   udpAddr,
			udpServer: udpServer,
		}

		if nil == udpServerConn {
		} // TODO
	}
}
