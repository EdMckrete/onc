package oncserver

import (
	"io"
	"net"
	"runtime"
	"sync"

	"github.com/swiftstack/onc"
	"github.com/swiftstack/xdr"
)

const (
	maxUdpPacketSize = uint32(65536) // Typically far less than this (e.g. ~2KB or ~7.5KB)
)

type serverProgVersSetStruct struct {
	minVers uint32
	maxVers uint32
	versMap map[uint32]bool // Map entry exists if the program supports the map key's version
}

type serverProgSetMap map[uint32]*serverProgVersSetStruct // Program Set (map entry exists if the map key's program is supported)

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
	recordFragmentMarkSize uint64
	tcpServerMap           map[uint16]*tcpServerStruct // Key is tcpServerStruct.port
	udpServerMap           map[uint16]*udpServerStruct // Key is udpServerStruct.port
}

var globals globalsStruct

func init() {
	var (
		err                error
		recordFragmentMark onc.RecordFragmentMarkStruct
	)

	globals.recordFragmentMarkSize, err = xdr.Examine(recordFragmentMark)
	if nil != err {
		panic(err)
	}

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
			tcpServer.tcpListenerHandlerExit()
		}

		tcpServerConn = &tcpServerConnStruct{
			tcpConn:   tcpConn,
			tcpServer: tcpServer,
		}

		tcpServerConn.WaitGroup.Add(1)

		go tcpServerConn.tcpConnHandler()
	}
}

func (tcpServer *tcpServerStruct) tcpListenerHandlerExit() {
	tcpServer.WaitGroup.Done()
	runtime.Goexit()
}

func (tcpServerConn *tcpServerConnStruct) tcpConnHandler() {
	var (
		bytesConsumed         uint64
		err                   error
		recordFragmentMark    onc.RecordFragmentMarkStruct
		recordFragmentMarkBuf []byte
		requestBuf            []byte
		requestBufSize        uint32
	)

	recordFragmentMarkBuf = make([]byte, globals.recordFragmentMarkSize, globals.recordFragmentMarkSize)

	for {
		_, err = io.ReadFull(tcpServerConn.tcpConn, recordFragmentMarkBuf)
		if nil != err {
			tcpServerConn.tcpConnHandlerExit()
		}

		bytesConsumed, err = xdr.Unpack(recordFragmentMarkBuf, &recordFragmentMark)
		if nil != err {
			tcpServerConn.tcpConnHandlerExit()
		}
		if bytesConsumed != globals.recordFragmentMarkSize {
			tcpServerConn.tcpConnHandlerExit()
		}
		if 1 != (recordFragmentMark.LastFragmentFlagAndFragmentLength >> 31) {
			tcpServerConn.tcpConnHandlerExit()
		}

		requestBufSize = recordFragmentMark.LastFragmentFlagAndFragmentLength & 0x7FFFFFFF
		requestBuf = make([]byte, requestBufSize, requestBufSize)

		_, err = io.ReadFull(tcpServerConn.tcpConn, requestBuf)
		if nil != err {
			tcpServerConn.tcpConnHandlerExit()
		}

		go requestHandler(requestBuf, tcpServerConn.tcpServer.serverProgSet, tcpServerConn.tcpServer.callbacks, tcpServerConn)
	}
}

func (tcpServerConn *tcpServerConnStruct) tcpConnHandlerExit() {
	tcpServerConn.WaitGroup.Done()
	runtime.Goexit()
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
			udpServer.udpConnHandlerExit()
		}

		udpPacketBuf = udpPacketBuf[:udpPacketLen]

		udpServerConn = &udpServerConnStruct{
			udpAddr:   udpAddr,
			udpServer: udpServer,
		}

		go requestHandler(udpPacketBuf, udpServer.serverProgSet, udpServer.callbacks, udpServerConn)
	}
}

func (udpServer *udpServerStruct) udpConnHandlerExit() {
	udpServer.WaitGroup.Done()
	runtime.Goexit()
}

func requestHandler(requestBuf []byte, serverProgSet serverProgSetMap, callbacks RequestCallbacks, connHandle ConnHandle) {
	var (
		authSysBody          *onc.AuthSysBodyStruct
		bytesConsumed        uint64
		err                  error
		ok                   bool
		rpcMsgCallBodyHeader onc.RpcMsgCallBodyHeaderStruct
		rpcMsgHeader         onc.RpcMsgHeaderStruct
		serverProgVersSet    *serverProgVersSetStruct
	)

	bytesConsumed, err = xdr.Unpack(requestBuf, &rpcMsgHeader)
	if nil != err {
		// Simply discard requestBuf that lacks a complete onc.RpcMsgHeaderStruct
		return
	}
	requestBuf = requestBuf[bytesConsumed:]

	if onc.Call != rpcMsgHeader.MType {
		// Simply discard requestBuf that is not an onc.Call
	}

	bytesConsumed, err = xdr.Unpack(requestBuf, &rpcMsgCallBodyHeader)
	if nil != err {
		// Simply discard requestBuf that lacks a complete onc. RpcMsgCallBodyHeaderStruct
		return
	}
	requestBuf = requestBuf[bytesConsumed:]

	if onc.RpcVers2 != rpcMsgCallBodyHeader.RpcVers {
		_ = SendDeniedRpcMismatchReply(connHandle, rpcMsgHeader.XID, onc.RpcVers2, onc.RpcVers2)
		return
	}

	if onc.AuthNone == rpcMsgCallBodyHeader.Cred.AuthFlavor {
		authSysBody = nil
	} else if onc.AuthSys == rpcMsgCallBodyHeader.Cred.AuthFlavor {
		authSysBody = &onc.AuthSysBodyStruct{}
		bytesConsumed, err = xdr.Unpack(rpcMsgCallBodyHeader.Cred.OpaqueBody, authSysBody)
		if nil != err {
			_ = SendDeniedAuthErrorReply(connHandle, rpcMsgHeader.XID, onc.AuthBadCred)
			return
		}
	} else {
		_ = SendDeniedAuthErrorReply(connHandle, rpcMsgHeader.XID, onc.AuthBadCred)
		return
	}

	if onc.AuthNone != rpcMsgCallBodyHeader.Verf.AuthFlavor {
		_ = SendDeniedAuthErrorReply(connHandle, rpcMsgHeader.XID, onc.AuthBadVerf)
		return
	}

	serverProgVersSet, ok = serverProgSet[rpcMsgCallBodyHeader.Prog]
	if !ok {
		_ = SendAcceptedOtherErrorReply(connHandle, rpcMsgHeader.XID, onc.ProgUnavail)
		return
	}

	_, ok = serverProgVersSet.versMap[rpcMsgCallBodyHeader.Vers]
	if !ok {
		_ = SendAcceptedProgMismatchReply(connHandle, rpcMsgHeader.XID, serverProgVersSet.minVers, serverProgVersSet.maxVers)
		return
	}

	callbacks.ONCRequest(connHandle, rpcMsgHeader.XID, rpcMsgCallBodyHeader.Prog, rpcMsgCallBodyHeader.Vers, rpcMsgCallBodyHeader.Proc, authSysBody, requestBuf)
}
