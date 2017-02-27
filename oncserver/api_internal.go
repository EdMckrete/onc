package oncserver

import (
	"net"
	"sync"
)

type serverProgVersSet map[uint32]bool // Program Version Set (map entry exists and is true if the program supports the map key's version)

type serverProgSet map[uint32]serverProgVersSet // Program Set (map entry exists if the map key's program is supported)

type tcpServerConnStruct struct {
	sync.Mutex
	tcpConn *net.TCPConn
}

type tcpServerStruct struct {
	sync.Mutex
	port             uint16
	sPS              serverProgSet
	tcpAddr          *net.TCPAddr
	tcpListener      *net.TCPListener
	tcpServerConnMap map[*net.TCPConn]*tcpServerConnStruct
}

type udpServerStruct struct {
	sync.Mutex
	port    uint16
	sPS     serverProgSet
	udpAddr *net.UDPAddr
	udpCon  *net.UDPConn
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

func startServer(prot uint32, port uint16, progVersList []ProgVersStruct, register bool, callbacks RequestCallbacks) (err error) {
	err = nil // TODO
	return
}

func stopServer(prot uint32, port uint16, deregister bool) (err error) {
	err = nil // TODO
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
