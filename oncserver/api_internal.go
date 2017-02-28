package oncserver

import (
	"fmt"
	"net"
	"strconv"

	"github.com/swiftstack/onc"
	"github.com/swiftstack/xdr"
)

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
			callbacks:        callbacks,
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
			callbacks:     callbacks,
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
	var (
		ok                                    bool
		recordFragmentMark                    onc.RecordFragmentMarkStruct
		recordFragmentMarkBuf                 []byte
		replyMsgBuf                           []byte
		replyMsgLength                        uint32
		rpcMsgHeader                          onc.RpcMsgHeaderStruct
		rpcMsgHeaderBuf                       []byte
		rpcMsgReplyBodyHeader                 onc.RpcMsgReplyBodyHeaderStruct
		rpcMsgReplyBodyHeaderBuf              []byte
		rpcMsgReplyBodyRejectedHeader         onc.RpcMsgReplyBodyRejectedHeaderStruct
		rpcMsgReplyBodyRejectedHeaderBuf      []byte
		rpcMsgReplyBodyRejectedRpcMismatch    onc.RpcMsgReplyBodyRejectedRpcMismatchStruct
		rpcMsgReplyBodyRejectedRpcMismatchBuf []byte
		tcpServerConn                         *tcpServerConnStruct
		udpServerConn                         *udpServerConnStruct
	)

	rpcMsgHeader.XID = xid
	rpcMsgHeader.MType = onc.Reply

	rpcMsgHeaderBuf, err = xdr.Pack(rpcMsgHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyHeader.Stat = onc.MsgDenied

	rpcMsgReplyBodyHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyRejectedHeader.Stat = onc.RpcMismatch

	rpcMsgReplyBodyRejectedHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyRejectedHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyRejectedRpcMismatch.MismatchInfoLow = mismatchInfoLow
	rpcMsgReplyBodyRejectedRpcMismatch.MismatchInfoHigh = mismatchInfoHigh

	rpcMsgReplyBodyRejectedRpcMismatchBuf, err = xdr.Pack(rpcMsgReplyBodyRejectedRpcMismatch)
	if nil != err {
		return
	}

	replyMsgLength = uint32(len(rpcMsgHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyRejectedHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyRejectedRpcMismatchBuf))

	tcpServerConn, ok = connHandle.(*tcpServerConnStruct)
	if ok {
		udpServerConn = nil
		recordFragmentMark.LastFragmentFlagAndFragmentLength = 0x80000000 | replyMsgLength

		recordFragmentMarkBuf, err = xdr.Pack(recordFragmentMark)
		if nil != err {
			return
		}

		replyMsgBuf = make([]byte, 0, uint32(len(recordFragmentMarkBuf))+replyMsgLength)

		replyMsgBuf = append(replyMsgBuf, recordFragmentMarkBuf...)
	} else {
		udpServerConn, ok = connHandle.(*udpServerConnStruct)
		if ok {
			replyMsgBuf = make([]byte, 0, replyMsgLength)
		} else {
			err = fmt.Errorf("connHandle must be one of *tcpServerConnStruct or *udpServerStruct")
			return
		}
	}

	replyMsgBuf = append(replyMsgBuf, rpcMsgHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyRejectedHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyRejectedRpcMismatchBuf...)

	if nil == udpServerConn {
		tcpServerConn.Lock()
		_, err = tcpServerConn.tcpConn.Write(replyMsgBuf)
		tcpServerConn.Unlock()
	} else {
		udpServerConn.udpServer.Lock()
		_, err = udpServerConn.udpServer.udpConn.WriteToUDP(replyMsgBuf, udpServerConn.udpAddr)
		udpServerConn.udpServer.Unlock()
	}

	return
}

func sendDeniedAuthErrorReply(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, stat uint32) (err error) {
	var (
		ok                                  bool
		recordFragmentMark                  onc.RecordFragmentMarkStruct
		recordFragmentMarkBuf               []byte
		replyMsgBuf                         []byte
		replyMsgLength                      uint32
		rpcMsgHeader                        onc.RpcMsgHeaderStruct
		rpcMsgHeaderBuf                     []byte
		rpcMsgReplyBodyHeader               onc.RpcMsgReplyBodyHeaderStruct
		rpcMsgReplyBodyHeaderBuf            []byte
		rpcMsgReplyBodyRejectedAuthError    onc.RpcMsgReplyBodyRejectedAuthErrorStruct
		rpcMsgReplyBodyRejectedAuthErrorBuf []byte
		rpcMsgReplyBodyRejectedHeader       onc.RpcMsgReplyBodyRejectedHeaderStruct
		rpcMsgReplyBodyRejectedHeaderBuf    []byte
		tcpServerConn                       *tcpServerConnStruct
		udpServerConn                       *udpServerConnStruct
	)

	rpcMsgHeader.XID = xid
	rpcMsgHeader.MType = onc.Reply

	rpcMsgHeaderBuf, err = xdr.Pack(rpcMsgHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyHeader.Stat = onc.MsgDenied

	rpcMsgReplyBodyHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyRejectedHeader.Stat = onc.AuthError

	rpcMsgReplyBodyRejectedHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyRejectedHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyRejectedAuthError.Stat = stat

	rpcMsgReplyBodyRejectedAuthErrorBuf, err = xdr.Pack(rpcMsgReplyBodyRejectedAuthError)
	if nil != err {
		return
	}

	replyMsgLength = uint32(len(rpcMsgHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyRejectedHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyRejectedAuthErrorBuf))

	tcpServerConn, ok = connHandle.(*tcpServerConnStruct)
	if ok {
		udpServerConn = nil
		recordFragmentMark.LastFragmentFlagAndFragmentLength = 0x80000000 | replyMsgLength

		recordFragmentMarkBuf, err = xdr.Pack(recordFragmentMark)
		if nil != err {
			return
		}

		replyMsgBuf = make([]byte, 0, uint32(len(recordFragmentMarkBuf))+replyMsgLength)

		replyMsgBuf = append(replyMsgBuf, recordFragmentMarkBuf...)
	} else {
		udpServerConn, ok = connHandle.(*udpServerConnStruct)
		if ok {
			replyMsgBuf = make([]byte, 0, replyMsgLength)
		} else {
			err = fmt.Errorf("connHandle must be one of *tcpServerConnStruct or *udpServerStruct")
			return
		}
	}

	replyMsgBuf = append(replyMsgBuf, rpcMsgHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyRejectedHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyRejectedAuthErrorBuf...)

	if nil == udpServerConn {
		tcpServerConn.Lock()
		_, err = tcpServerConn.tcpConn.Write(replyMsgBuf)
		tcpServerConn.Unlock()
	} else {
		udpServerConn.udpServer.Lock()
		_, err = udpServerConn.udpServer.udpConn.WriteToUDP(replyMsgBuf, udpServerConn.udpAddr)
		udpServerConn.udpServer.Unlock()
	}

	return
}

func sendAcceptedSuccess(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, results []byte) (err error) {
	var (
		ok                               bool
		recordFragmentMark               onc.RecordFragmentMarkStruct
		recordFragmentMarkBuf            []byte
		replyMsgBuf                      []byte
		replyMsgLength                   uint32
		rpcMsgHeader                     onc.RpcMsgHeaderStruct
		rpcMsgHeaderBuf                  []byte
		rpcMsgReplyBodyHeader            onc.RpcMsgReplyBodyHeaderStruct
		rpcMsgReplyBodyHeaderBuf         []byte
		rpcMsgReplyBodyAcceptedHeader    onc.RpcMsgReplyBodyAcceptedHeaderStruct
		rpcMsgReplyBodyAcceptedHeaderBuf []byte
		tcpServerConn                    *tcpServerConnStruct
		udpServerConn                    *udpServerConnStruct
	)

	rpcMsgHeader.XID = xid
	rpcMsgHeader.MType = onc.Reply

	rpcMsgHeaderBuf, err = xdr.Pack(rpcMsgHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyHeader.Stat = onc.MsgAccepted

	rpcMsgReplyBodyHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyAcceptedHeader.Verf.AuthFlavor = onc.AuthNone
	rpcMsgReplyBodyAcceptedHeader.Verf.OpaqueBody = []byte{}

	rpcMsgReplyBodyAcceptedHeader.Stat = onc.Success

	rpcMsgReplyBodyAcceptedHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyAcceptedHeader)
	if nil != err {
		return
	}

	replyMsgLength = uint32(len(rpcMsgHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyAcceptedHeaderBuf))
	replyMsgLength += uint32(len(results))

	tcpServerConn, ok = connHandle.(*tcpServerConnStruct)
	if ok {
		udpServerConn = nil
		recordFragmentMark.LastFragmentFlagAndFragmentLength = 0x80000000 | replyMsgLength

		recordFragmentMarkBuf, err = xdr.Pack(recordFragmentMark)
		if nil != err {
			return
		}

		replyMsgBuf = make([]byte, 0, uint32(len(recordFragmentMarkBuf))+replyMsgLength)

		replyMsgBuf = append(replyMsgBuf, recordFragmentMarkBuf...)
	} else {
		udpServerConn, ok = connHandle.(*udpServerConnStruct)
		if ok {
			replyMsgBuf = make([]byte, 0, replyMsgLength)
		} else {
			err = fmt.Errorf("connHandle must be one of *tcpServerConnStruct or *udpServerStruct")
			return
		}
	}

	replyMsgBuf = append(replyMsgBuf, rpcMsgHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyAcceptedHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, results...)

	if nil == udpServerConn {
		tcpServerConn.Lock()
		_, err = tcpServerConn.tcpConn.Write(replyMsgBuf)
		tcpServerConn.Unlock()
	} else {
		udpServerConn.udpServer.Lock()
		_, err = udpServerConn.udpServer.udpConn.WriteToUDP(replyMsgBuf, udpServerConn.udpAddr)
		udpServerConn.udpServer.Unlock()
	}

	return
}

func sendAcceptedProgMismatchReply(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, mismatchInfoLow uint32, mismatchInfoHigh uint32) (err error) {
	var (
		ok                                    bool
		recordFragmentMark                    onc.RecordFragmentMarkStruct
		recordFragmentMarkBuf                 []byte
		replyMsgLength                        uint32
		replyMsgBuf                           []byte
		rpcMsgHeader                          onc.RpcMsgHeaderStruct
		rpcMsgHeaderBuf                       []byte
		rpcMsgReplyBodyHeader                 onc.RpcMsgReplyBodyHeaderStruct
		rpcMsgReplyBodyHeaderBuf              []byte
		rpcMsgReplyBodyAcceptedHeader         onc.RpcMsgReplyBodyAcceptedHeaderStruct
		rpcMsgReplyBodyAcceptedHeaderBuf      []byte
		pcMsgReplyBodyAcceptedProgMismatch    onc.RpcMsgReplyBodyAcceptedProgMismatchStruct
		pcMsgReplyBodyAcceptedProgMismatchBuf []byte
		tcpServerConn                         *tcpServerConnStruct
		udpServerConn                         *udpServerConnStruct
	)

	rpcMsgHeader.XID = xid
	rpcMsgHeader.MType = onc.Reply

	rpcMsgHeaderBuf, err = xdr.Pack(rpcMsgHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyHeader.Stat = onc.MsgAccepted

	rpcMsgReplyBodyHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyAcceptedHeader.Verf.AuthFlavor = onc.AuthNone
	rpcMsgReplyBodyAcceptedHeader.Verf.OpaqueBody = []byte{}

	rpcMsgReplyBodyAcceptedHeader.Stat = onc.ProgMismatch

	rpcMsgReplyBodyAcceptedHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyAcceptedHeader)
	if nil != err {
		return
	}

	pcMsgReplyBodyAcceptedProgMismatch.MismatchInfoLow = mismatchInfoLow
	pcMsgReplyBodyAcceptedProgMismatch.MismatchInfoHigh = mismatchInfoHigh

	pcMsgReplyBodyAcceptedProgMismatchBuf, err = xdr.Pack(pcMsgReplyBodyAcceptedProgMismatch)
	if nil != err {
		return
	}

	replyMsgLength = uint32(len(rpcMsgHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyAcceptedHeaderBuf))
	replyMsgLength += uint32(len(pcMsgReplyBodyAcceptedProgMismatchBuf))

	tcpServerConn, ok = connHandle.(*tcpServerConnStruct)
	if ok {
		udpServerConn = nil
		recordFragmentMark.LastFragmentFlagAndFragmentLength = 0x80000000 | replyMsgLength

		recordFragmentMarkBuf, err = xdr.Pack(recordFragmentMark)
		if nil != err {
			return
		}

		replyMsgBuf = make([]byte, 0, uint32(len(recordFragmentMarkBuf))+replyMsgLength)

		replyMsgBuf = append(replyMsgBuf, recordFragmentMarkBuf...)
	} else {
		udpServerConn, ok = connHandle.(*udpServerConnStruct)
		if ok {
			replyMsgBuf = make([]byte, 0, replyMsgLength)
		} else {
			err = fmt.Errorf("connHandle must be one of *tcpServerConnStruct or *udpServerStruct")
			return
		}
	}

	replyMsgBuf = append(replyMsgBuf, rpcMsgHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyAcceptedHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, pcMsgReplyBodyAcceptedProgMismatchBuf...)

	if nil == udpServerConn {
		tcpServerConn.Lock()
		_, err = tcpServerConn.tcpConn.Write(replyMsgBuf)
		tcpServerConn.Unlock()
	} else {
		udpServerConn.udpServer.Lock()
		_, err = udpServerConn.udpServer.udpConn.WriteToUDP(replyMsgBuf, udpServerConn.udpAddr)
		udpServerConn.udpServer.Unlock()
	}

	return
}

func sendAcceptedOtherErrorReply(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, stat uint32) (err error) {
	var (
		ok                               bool
		recordFragmentMark               onc.RecordFragmentMarkStruct
		recordFragmentMarkBuf            []byte
		replyMsgLength                   uint32
		replyMsgBuf                      []byte
		rpcMsgHeader                     onc.RpcMsgHeaderStruct
		rpcMsgHeaderBuf                  []byte
		rpcMsgReplyBodyHeader            onc.RpcMsgReplyBodyHeaderStruct
		rpcMsgReplyBodyHeaderBuf         []byte
		rpcMsgReplyBodyAcceptedHeader    onc.RpcMsgReplyBodyAcceptedHeaderStruct
		rpcMsgReplyBodyAcceptedHeaderBuf []byte
		tcpServerConn                    *tcpServerConnStruct
		udpServerConn                    *udpServerConnStruct
	)

	rpcMsgHeader.XID = xid
	rpcMsgHeader.MType = onc.Reply

	rpcMsgHeaderBuf, err = xdr.Pack(rpcMsgHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyHeader.Stat = onc.MsgAccepted

	rpcMsgReplyBodyHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyHeader)
	if nil != err {
		return
	}

	rpcMsgReplyBodyAcceptedHeader.Verf.AuthFlavor = onc.AuthNone
	rpcMsgReplyBodyAcceptedHeader.Verf.OpaqueBody = []byte{}

	rpcMsgReplyBodyAcceptedHeader.Stat = stat

	rpcMsgReplyBodyAcceptedHeaderBuf, err = xdr.Pack(rpcMsgReplyBodyAcceptedHeader)
	if nil != err {
		return
	}

	replyMsgLength = uint32(len(rpcMsgHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyHeaderBuf))
	replyMsgLength += uint32(len(rpcMsgReplyBodyAcceptedHeaderBuf))

	tcpServerConn, ok = connHandle.(*tcpServerConnStruct)
	if ok {
		udpServerConn = nil
		recordFragmentMark.LastFragmentFlagAndFragmentLength = 0x80000000 | replyMsgLength

		recordFragmentMarkBuf, err = xdr.Pack(recordFragmentMark)
		if nil != err {
			return
		}

		replyMsgBuf = make([]byte, 0, uint32(len(recordFragmentMarkBuf))+replyMsgLength)

		replyMsgBuf = append(replyMsgBuf, recordFragmentMarkBuf...)
	} else {
		udpServerConn, ok = connHandle.(*udpServerConnStruct)
		if ok {
			replyMsgBuf = make([]byte, 0, replyMsgLength)
		} else {
			err = fmt.Errorf("connHandle must be one of *tcpServerConnStruct or *udpServerStruct")
			return
		}
	}

	replyMsgBuf = append(replyMsgBuf, rpcMsgHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyHeaderBuf...)
	replyMsgBuf = append(replyMsgBuf, rpcMsgReplyBodyAcceptedHeaderBuf...)

	if nil == udpServerConn {
		tcpServerConn.Lock()
		_, err = tcpServerConn.tcpConn.Write(replyMsgBuf)
		tcpServerConn.Unlock()
	} else {
		udpServerConn.udpServer.Lock()
		_, err = udpServerConn.udpServer.udpConn.WriteToUDP(replyMsgBuf, udpServerConn.udpAddr)
		udpServerConn.udpServer.Unlock()
	}

	return
}
