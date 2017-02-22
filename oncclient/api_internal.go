package oncclient

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"sync"

	"github.com/swiftstack/onc"
	"github.com/swiftstack/xdr"
)

type tcpConnContextStruct struct {
	sync.Mutex        //                           Protects tcpConn & replyCallbacksMap
	tcpConn           *net.TCPConn
	replyCallbacksMap map[uint32]ReplyCallbacks // Key == XID; Value == Call using that XID's callback interface
}

type udpConnContextStruct struct {
	sync.Mutex        //                           Protects udpConn & replyCallbacksMap
	udpConn           *net.UDPConn
	replyCallbacksMap map[uint32]ReplyCallbacks // Key == XID; Value == Call using that XID's callback interface
}

type globalsStruct struct {
	sync.Mutex // Protects nextXID, tcpConnContextMap, & udpConnContextMap
	sync.WaitGroup
	recordFragmentMarkSize uint64
	nextXID                uint32
	tcpConnContextMap      map[*net.TCPConn]*tcpConnContextStruct
	udpConnContextMap      map[*net.UDPConn]*udpConnContextStruct
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

	globals.nextXID = 0
	globals.tcpConnContextMap = make(map[*net.TCPConn]*tcpConnContextStruct)
	globals.udpConnContextMap = make(map[*net.UDPConn]*udpConnContextStruct)
}

func fetchXID() (xid uint32) {
	globals.Lock()
	xid = globals.nextXID
	globals.nextXID++
	globals.Unlock()
	return
}

func issueTCPCall(tcpConn *net.TCPConn, prog uint32, vers uint32, proc uint32, authSysBody *onc.AuthSysBodyStruct, procArgs interface{}, replyCallbacks ReplyCallbacks) (err error) {
	var (
		authSysBodyBuf              []byte
		callBuf                     []byte
		callRecordFragmentMark      onc.RecordFragmentMarkStruct
		callRecordFragmentMarkBuf   []byte
		callRpcMsgCallBodyHeader    onc.RpcMsgCallBodyHeaderStruct
		callRpcMsgCallBodyHeaderBuf []byte
		callRpcMsgHeader            onc.RpcMsgHeaderStruct
		callRpcMsgHeaderBuf         []byte
		ok                          bool
		procArgsBuf                 []byte
		tcpConnContext              *tcpConnContextStruct
		xid                         uint32
	)

	xid = fetchXID()

	callRpcMsgHeader = onc.RpcMsgHeaderStruct{
		XID:   xid,
		MType: onc.Call,
	}

	callRpcMsgHeaderBuf, err = xdr.Pack(callRpcMsgHeader)
	if nil != err {
		return
	}

	if nil == authSysBody {
		callRpcMsgCallBodyHeader = onc.RpcMsgCallBodyHeaderStruct{
			RpcVers: onc.RpcVers2,
			Prog:    prog,
			Vers:    vers,
			Proc:    proc,
			Cred: onc.OpaqueAuthStruct{
				AuthFlavor: onc.AuthNone,
				OpaqueBody: []byte{},
			},
			Verf: onc.OpaqueAuthStruct{
				AuthFlavor: onc.AuthNone,
				OpaqueBody: []byte{},
			},
		}
	} else {
		authSysBodyBuf, err = xdr.Pack(authSysBody)
		if nil != err {
			return
		}

		callRpcMsgCallBodyHeader = onc.RpcMsgCallBodyHeaderStruct{
			RpcVers: onc.RpcVers2,
			Prog:    prog,
			Vers:    vers,
			Proc:    proc,
			Cred: onc.OpaqueAuthStruct{
				AuthFlavor: onc.AuthSys,
				OpaqueBody: authSysBodyBuf,
			},
			Verf: onc.OpaqueAuthStruct{
				AuthFlavor: onc.AuthNone,
				OpaqueBody: []byte{},
			},
		}
	}

	callRpcMsgCallBodyHeaderBuf, err = xdr.Pack(callRpcMsgCallBodyHeader)
	if nil != err {
		return
	}

	if nil == procArgs {
		callRecordFragmentMark = onc.RecordFragmentMarkStruct{
			LastFragmentFlagAndFragmentLength: 0x80000000 | uint32(len(callRpcMsgHeaderBuf)+len(callRpcMsgCallBodyHeaderBuf)),
		}

		callBuf = make([]byte, 0, len(callRpcMsgHeaderBuf)+len(callRpcMsgCallBodyHeaderBuf))
	} else {
		procArgsBuf, err = xdr.Pack(procArgs)
		if nil != err {
			return
		}

		callRecordFragmentMark = onc.RecordFragmentMarkStruct{
			LastFragmentFlagAndFragmentLength: 0x80000000 | uint32(len(callRpcMsgHeaderBuf)+len(callRpcMsgCallBodyHeaderBuf)+len(procArgsBuf)),
		}

		callBuf = make([]byte, 0, len(callRpcMsgHeaderBuf)+len(callRpcMsgCallBodyHeaderBuf)+len(procArgsBuf))
	}

	callRecordFragmentMarkBuf, err = xdr.Pack(callRecordFragmentMark)
	if nil != err {
		return
	}

	callBuf = append(callBuf, callRecordFragmentMarkBuf...)
	callBuf = append(callBuf, callRpcMsgHeaderBuf...)
	callBuf = append(callBuf, callRpcMsgCallBodyHeaderBuf...)

	if nil != procArgs {
		callBuf = append(callBuf, procArgsBuf...)
	}

	globals.Lock()

	tcpConnContext, ok = globals.tcpConnContextMap[tcpConn]

	if ok {
		tcpConnContext.Lock()
		globals.Unlock()
	} else {
		tcpConnContext = &tcpConnContextStruct{
			tcpConn:           tcpConn,
			replyCallbacksMap: make(map[uint32]ReplyCallbacks),
		}

		tcpConnContext.Lock()

		globals.Add(1)

		go tcpConnContext.replyHandler()

		globals.Unlock()
	}

	tcpConnContext.replyCallbacksMap[xid] = replyCallbacks

	_, err = io.CopyN(tcpConn, bytes.NewReader(callBuf), int64(len(callBuf)))
	if nil != err {
		tcpConnContext.Unlock()
		return
	}

	tcpConnContext.Unlock()

	err = nil
	return
}

func (tCCS *tcpConnContextStruct) replyHandler() {
	var (
		bytesConsumed              uint64
		err                        error
		ok                         bool
		replyBuf                   []byte
		replyBufSize               uint32
		replyCallbacks             ReplyCallbacks
		replyRecordFragmentMark    onc.RecordFragmentMarkStruct
		replyRecordFragmentMarkBuf []byte
		rpcMsgHeader               onc.RpcMsgHeaderStruct
	)

	replyRecordFragmentMarkBuf = make([]byte, globals.recordFragmentMarkSize, globals.recordFragmentMarkSize)

	for {
		_, err = io.ReadFull(tCCS.tcpConn, replyRecordFragmentMarkBuf)
		if nil != err {
			tCCS.replyHandlerExit()
		}

		bytesConsumed, err = xdr.Unpack(replyRecordFragmentMarkBuf, &replyRecordFragmentMark)
		if nil != err {
			tCCS.replyHandlerExit()
		}
		if bytesConsumed != globals.recordFragmentMarkSize {
			tCCS.replyHandlerExit()
		}
		if 1 != (replyRecordFragmentMark.LastFragmentFlagAndFragmentLength >> 31) {
			tCCS.replyHandlerExit()
		}

		replyBufSize = replyRecordFragmentMark.LastFragmentFlagAndFragmentLength & 0x7FFFFFFF
		replyBuf = make([]byte, replyBufSize, replyBufSize)

		_, err = io.ReadFull(tCCS.tcpConn, replyBuf)
		if nil != err {
			tCCS.replyHandlerExit()
		}

		bytesConsumed, err = xdr.Unpack(replyBuf, &rpcMsgHeader)
		if nil != err {
			tCCS.replyHandlerExit()
		}
		if onc.Reply != rpcMsgHeader.MType {
			tCCS.replyHandlerExit()
		}

		tCCS.Lock()
		replyCallbacks, ok = tCCS.replyCallbacksMap[rpcMsgHeader.XID]
		if !ok {
			tCCS.Unlock()
			tCCS.replyHandlerExit()
		}
		delete(tCCS.replyCallbacksMap, rpcMsgHeader.XID)
		tCCS.Unlock()

		ok = replyHandlerProcessPayload(replyBuf[bytesConsumed:], replyCallbacks)
		if !ok {
			tCCS.replyHandlerExit()
		}
	}
}

func (tCCS *tcpConnContextStruct) replyHandlerExit() {
	globals.Lock()
	delete(globals.tcpConnContextMap, tCCS.tcpConn)
	tCCS.Lock()
	_ = tCCS.tcpConn.Close()
	for _, replyCallbacks := range tCCS.replyCallbacksMap {
		replyCallbacks.ONCReplyConnectionDown()
	}
	tCCS.Unlock()
	globals.Unlock()
	globals.Done()
	runtime.Goexit()
}

func issueUDPCall(udpConn *net.UDPConn, prog uint32, vers uint32, proc uint32, authSysBody *onc.AuthSysBodyStruct, procArgs interface{}, replyCallbacks ReplyCallbacks) (err error) {
	var (
		authSysBodyBuf              []byte
		bytesWritten                int
		callBuf                     []byte
		callRpcMsgCallBodyHeader    onc.RpcMsgCallBodyHeaderStruct
		callRpcMsgCallBodyHeaderBuf []byte
		callRpcMsgHeader            onc.RpcMsgHeaderStruct
		callRpcMsgHeaderBuf         []byte
		ok                          bool
		procArgsBuf                 []byte
		udpConnContext              *udpConnContextStruct
		xid                         uint32
	)

	xid = fetchXID()

	callRpcMsgHeader = onc.RpcMsgHeaderStruct{
		XID:   xid,
		MType: onc.Call,
	}

	callRpcMsgHeaderBuf, err = xdr.Pack(callRpcMsgHeader)
	if nil != err {
		return
	}

	if nil == authSysBody {
		callRpcMsgCallBodyHeader = onc.RpcMsgCallBodyHeaderStruct{
			RpcVers: onc.RpcVers2,
			Prog:    prog,
			Vers:    vers,
			Proc:    proc,
			Cred: onc.OpaqueAuthStruct{
				AuthFlavor: onc.AuthNone,
				OpaqueBody: []byte{},
			},
			Verf: onc.OpaqueAuthStruct{
				AuthFlavor: onc.AuthNone,
				OpaqueBody: []byte{},
			},
		}
	} else {
		authSysBodyBuf, err = xdr.Pack(authSysBody)
		if nil != err {
			return
		}

		callRpcMsgCallBodyHeader = onc.RpcMsgCallBodyHeaderStruct{
			RpcVers: onc.RpcVers2,
			Prog:    prog,
			Vers:    vers,
			Proc:    proc,
			Cred: onc.OpaqueAuthStruct{
				AuthFlavor: onc.AuthSys,
				OpaqueBody: authSysBodyBuf,
			},
			Verf: onc.OpaqueAuthStruct{
				AuthFlavor: onc.AuthNone,
				OpaqueBody: []byte{},
			},
		}
	}

	callRpcMsgCallBodyHeaderBuf, err = xdr.Pack(callRpcMsgCallBodyHeader)
	if nil != err {
		return
	}

	if nil == procArgs {
		callBuf = make([]byte, 0, len(callRpcMsgHeaderBuf)+len(callRpcMsgCallBodyHeaderBuf))
	} else {
		procArgsBuf, err = xdr.Pack(procArgs)
		if nil != err {
			return
		}

		callBuf = make([]byte, 0, len(callRpcMsgHeaderBuf)+len(callRpcMsgCallBodyHeaderBuf)+len(procArgsBuf))
	}

	callBuf = append(callBuf, callRpcMsgHeaderBuf...)
	callBuf = append(callBuf, callRpcMsgCallBodyHeaderBuf...)

	if nil != procArgs {
		callBuf = append(callBuf, procArgsBuf...)
	}

	globals.Lock()

	udpConnContext, ok = globals.udpConnContextMap[udpConn]

	if ok {
		udpConnContext.Lock()
		globals.Unlock()
	} else {
		udpConnContext = &udpConnContextStruct{
			udpConn:           udpConn,
			replyCallbacksMap: make(map[uint32]ReplyCallbacks),
		}

		udpConnContext.Lock()

		globals.Add(1)

		go udpConnContext.replyHandler()

		globals.Unlock()
	}

	udpConnContext.replyCallbacksMap[xid] = replyCallbacks

	bytesWritten, err = udpConn.Write(callBuf)
	if nil != err {
		udpConnContext.Unlock()
		return
	}
	if len(callBuf) != bytesWritten {
		udpConnContext.Unlock()
		err = fmt.Errorf("unable to write callBuf to UDP socket")
		return
	}

	udpConnContext.Unlock()

	err = nil
	return
}

func (uCCS *udpConnContextStruct) replyHandler() {
	var (
		bytesConsumed  uint64
		bytesRead      int
		err            error
		ok             bool
		replyBuf       []byte
		replyCallbacks ReplyCallbacks
		rpcMsgHeader   onc.RpcMsgHeaderStruct
	)

	for {
		replyBuf = make([]byte, onc.MaxUDPPacketSize, onc.MaxUDPPacketSize)

		bytesRead, err = uCCS.udpConn.Read(replyBuf)
		if nil != err {
			uCCS.replyHandlerExit()
		}
		if 0 == bytesRead {
			uCCS.replyHandlerExit()
		}

		replyBuf = replyBuf[:bytesRead]

		bytesConsumed, err = xdr.Unpack(replyBuf, &rpcMsgHeader)
		if nil != err {
			uCCS.replyHandlerExit()
		}
		if onc.Reply != rpcMsgHeader.MType {
			uCCS.replyHandlerExit()
		}

		uCCS.Lock()
		replyCallbacks, ok = uCCS.replyCallbacksMap[rpcMsgHeader.XID]
		if !ok {
			uCCS.Unlock()
			uCCS.replyHandlerExit()
		}
		delete(uCCS.replyCallbacksMap, rpcMsgHeader.XID)
		uCCS.Unlock()

		ok = replyHandlerProcessPayload(replyBuf[bytesConsumed:], replyCallbacks)
		if !ok {
			uCCS.replyHandlerExit()
		}
	}
}

func (uCCS *udpConnContextStruct) replyHandlerExit() {
	globals.Lock()
	delete(globals.udpConnContextMap, uCCS.udpConn)
	uCCS.Lock()
	_ = uCCS.udpConn.Close()
	for _, replyCallbacks := range uCCS.replyCallbacksMap {
		replyCallbacks.ONCReplyConnectionDown()
	}
	uCCS.Unlock()
	globals.Unlock()
	globals.Done()
	runtime.Goexit()
}

func replyHandlerProcessPayload(rpcMsgReplyBodyHeaderBuf []byte, replyCallbacks ReplyCallbacks) (ok bool) {
	var (
		bytesConsumed                          uint64
		err                                    error
		results                                []byte
		rpcMsgReplyBodyAcceptedHeader          onc.RpcMsgReplyBodyAcceptedHeaderStruct
		rpcMsgReplyBodyAcceptedHeaderBuf       []byte
		rpcMsgReplyBodyAcceptedProgMismatch    onc.RpcMsgReplyBodyAcceptedProgMismatchStruct
		rpcMsgReplyBodyAcceptedProgMismatchBuf []byte
		rpcMsgReplyBodyHeader                  onc.RpcMsgReplyBodyHeaderStruct
		rpcMsgReplyBodyRejectedAuthError       onc.RpcMsgReplyBodyRejectedAuthErrorStruct
		rpcMsgReplyBodyRejectedAuthErrorBuf    []byte
		rpcMsgReplyBodyRejectedHeader          onc.RpcMsgReplyBodyRejectedHeaderStruct
		rpcMsgReplyBodyRejectedHeaderBuf       []byte
		rpcMsgReplyBodyRejectedRpcMismatch     onc.RpcMsgReplyBodyRejectedRpcMismatchStruct
		rpcMsgReplyBodyRejectedRpcMismatchBuf  []byte
	)

	bytesConsumed, err = xdr.Unpack(rpcMsgReplyBodyHeaderBuf, &rpcMsgReplyBodyHeader)
	if nil != err {
		replyCallbacks.ONCReplyConnectionDown()
		ok = false
		return
	}

	switch rpcMsgReplyBodyHeader.Stat {
	case onc.MsgAccepted:
		rpcMsgReplyBodyAcceptedHeaderBuf = rpcMsgReplyBodyHeaderBuf[bytesConsumed:]

		bytesConsumed, err = xdr.Unpack(rpcMsgReplyBodyAcceptedHeaderBuf, &rpcMsgReplyBodyAcceptedHeader)
		if nil != err {
			replyCallbacks.ONCReplyConnectionDown()
			ok = false
			return
		}
		if onc.AuthNone != rpcMsgReplyBodyAcceptedHeader.Verf.AuthFlavor {
			replyCallbacks.ONCReplyConnectionDown()
			ok = false
			return
		}

		switch rpcMsgReplyBodyAcceptedHeader.Stat {
		case onc.Success:
			results = rpcMsgReplyBodyAcceptedHeaderBuf[bytesConsumed:]

			replyCallbacks.ONCReplySuccess(results)
		case onc.ProgMismatch:
			rpcMsgReplyBodyAcceptedProgMismatchBuf = rpcMsgReplyBodyAcceptedHeaderBuf[bytesConsumed:]

			_, err = xdr.Unpack(rpcMsgReplyBodyAcceptedProgMismatchBuf, &rpcMsgReplyBodyAcceptedProgMismatch)
			if nil != err {
				replyCallbacks.ONCReplyConnectionDown()
				ok = false
				return
			}

			replyCallbacks.ONCReplyProgMismatch(rpcMsgReplyBodyAcceptedProgMismatch.MismatchInfoLow, rpcMsgReplyBodyAcceptedProgMismatch.MismatchInfoHigh)
		default:
			replyCallbacks.ONCReplyConnectionDown()
			ok = false
			return
		}
	case onc.MsgDenied:
		rpcMsgReplyBodyRejectedHeaderBuf = rpcMsgReplyBodyHeaderBuf[bytesConsumed:]

		bytesConsumed, err = xdr.Unpack(rpcMsgReplyBodyRejectedHeaderBuf, &rpcMsgReplyBodyRejectedHeader)
		if nil != err {
			replyCallbacks.ONCReplyConnectionDown()
			ok = false
			return
		}

		switch rpcMsgReplyBodyRejectedHeader.Stat {
		case onc.RpcMismatch:
			rpcMsgReplyBodyRejectedRpcMismatchBuf = rpcMsgReplyBodyRejectedHeaderBuf[bytesConsumed:]

			_, err = xdr.Unpack(rpcMsgReplyBodyRejectedRpcMismatchBuf, &rpcMsgReplyBodyRejectedRpcMismatch)
			if nil != err {
				replyCallbacks.ONCReplyConnectionDown()
				ok = false
				return
			}

			replyCallbacks.ONCReplyRpcMismatch(rpcMsgReplyBodyRejectedRpcMismatch.MismatchInfoLow, rpcMsgReplyBodyRejectedRpcMismatch.MismatchInfoHigh)
		case onc.AuthError:
			rpcMsgReplyBodyRejectedAuthErrorBuf = rpcMsgReplyBodyRejectedHeaderBuf[bytesConsumed:]

			_, err = xdr.Unpack(rpcMsgReplyBodyRejectedAuthErrorBuf, &rpcMsgReplyBodyRejectedAuthError)
			if nil != err {
				replyCallbacks.ONCReplyConnectionDown()
				ok = false
				return
			}

			replyCallbacks.ONCReplyAuthError(rpcMsgReplyBodyRejectedAuthError.Stat)
		default:
			replyCallbacks.ONCReplyConnectionDown()
			ok = false
			return
		}
	default:
		ok = false
		return
	}

	ok = true
	return
}

func locateService(addr string, prog uint32, vers uint32, prot uint32) (port uint16, err error) {
	var (
		bytesConsumed                 uint64
		bytesRead                     int
		bytesWritten                  int
		callBuf                       []byte
		callPmapMapping               onc.PmapMappingStruct
		callPmapMappingBuf            []byte
		callRecordFragmentMark        onc.RecordFragmentMarkStruct
		callRecordFragmentMarkBuf     []byte
		callRpcMsgCallBodyHeader      onc.RpcMsgCallBodyHeaderStruct
		callRpcMsgCallBodyHeaderBuf   []byte
		callRpcMsgHeader              onc.RpcMsgHeaderStruct
		callRpcMsgHeaderBuf           []byte
		replyBuf                      []byte
		replyRecordFragmentMark       onc.RecordFragmentMarkStruct
		replyRecordFragmentMarkBuf    []byte
		replyRecordFragmentMarkBufLen uint64
		tcpAddr                       *net.TCPAddr
		tcpConn                       *net.TCPConn
		udpAddr                       *net.UDPAddr
		udpConn                       *net.UDPConn
		xid                           uint32
	)

	xid = fetchXID()

	callRpcMsgHeader = onc.RpcMsgHeaderStruct{
		XID:   xid,
		MType: onc.Call,
	}

	callRpcMsgHeaderBuf, err = xdr.Pack(callRpcMsgHeader)
	if nil != err {
		return
	}

	callRpcMsgCallBodyHeader = onc.RpcMsgCallBodyHeaderStruct{
		RpcVers: onc.RpcVers2,
		Prog:    onc.ProgNumPortMap,
		Vers:    onc.PmapVers,
		Proc:    onc.PmapProcGetAddr,
		Cred: onc.OpaqueAuthStruct{
			AuthFlavor: onc.AuthNone,
			OpaqueBody: []byte{},
		},
		Verf: onc.OpaqueAuthStruct{
			AuthFlavor: onc.AuthNone,
			OpaqueBody: []byte{},
		},
	}

	callRpcMsgCallBodyHeaderBuf, err = xdr.Pack(callRpcMsgCallBodyHeader)
	if nil != err {
		return
	}

	callPmapMapping = onc.PmapMappingStruct{
		Prog: prog,
		Vers: vers,
		Prot: prot,
		Port: 0,
	}

	callPmapMappingBuf, err = xdr.Pack(callPmapMapping)
	if nil != err {
		return
	}

	switch prot {
	case onc.IPProtoTCP:
		callRecordFragmentMark = onc.RecordFragmentMarkStruct{
			LastFragmentFlagAndFragmentLength: 0x80000000 | uint32(len(callRpcMsgHeaderBuf)+len(callRpcMsgCallBodyHeaderBuf)+len(callPmapMappingBuf)),
		}

		callRecordFragmentMarkBuf, err = xdr.Pack(callRecordFragmentMark)
		if nil != err {
			return
		}

		callBuf = make([]byte, 0, len(callRecordFragmentMarkBuf)+len(callRpcMsgHeaderBuf)+len(callRpcMsgCallBodyHeaderBuf)+len(callPmapMappingBuf))
		callBuf = append(callBuf, callRecordFragmentMarkBuf...)
		callBuf = append(callBuf, callRpcMsgHeaderBuf...)
		callBuf = append(callBuf, callRpcMsgCallBodyHeaderBuf...)
		callBuf = append(callBuf, callPmapMappingBuf...)

		tcpAddr, err = net.ResolveTCPAddr("tcp4", addr+":"+strconv.FormatUint(uint64(onc.PmapAndRpcbPort), 10))
		if nil != err {
			return
		}

		tcpConn, err = net.DialTCP("tcp4", nil, tcpAddr)
		if nil != err {
			return
		}

		_, err = io.CopyN(tcpConn, bytes.NewReader(callBuf), int64(len(callBuf)))
		if nil != err {
			_ = tcpConn.Close()
			return
		}

		replyRecordFragmentMarkBufLen, err = xdr.Examine(replyRecordFragmentMark)
		if nil != err {
			_ = tcpConn.Close()
			return
		}

		replyRecordFragmentMarkBuf = make([]byte, replyRecordFragmentMarkBufLen, replyRecordFragmentMarkBufLen)

		_, err = io.ReadFull(tcpConn, replyRecordFragmentMarkBuf)
		if nil != err {
			_ = tcpConn.Close()
			return
		}

		bytesConsumed, err = xdr.Unpack(replyRecordFragmentMarkBuf, &replyRecordFragmentMark)
		if nil != err {
			_ = tcpConn.Close()
			return
		}
		if uint64(len(replyRecordFragmentMarkBuf)) != bytesConsumed {
			_ = tcpConn.Close()
			err = fmt.Errorf("problem unpacking replyRecordFragmentMarkBuf")
			return
		}
		if 1 != (replyRecordFragmentMark.LastFragmentFlagAndFragmentLength >> 31) {
			_ = tcpConn.Close()
			err = fmt.Errorf("multi-fragment records not supported")
			return
		}

		replyBuf = make([]byte, (replyRecordFragmentMark.LastFragmentFlagAndFragmentLength & 0x7FFFFFFF), (replyRecordFragmentMark.LastFragmentFlagAndFragmentLength & 0x7FFFFFFF))

		_, err = io.ReadFull(tcpConn, replyBuf)
		if nil != err {
			_ = tcpConn.Close()
			return
		}

		err = tcpConn.Close()
		if nil != err {
			return
		}
	case onc.IPProtoUDP:
		callBuf = make([]byte, 0, len(callRpcMsgHeaderBuf)+len(callRpcMsgCallBodyHeaderBuf)+len(callPmapMappingBuf))
		callBuf = append(callBuf, callRpcMsgHeaderBuf...)
		callBuf = append(callBuf, callRpcMsgCallBodyHeaderBuf...)
		callBuf = append(callBuf, callPmapMappingBuf...)

		udpAddr, err = net.ResolveUDPAddr("udp4", addr+":"+strconv.FormatUint(uint64(onc.PmapAndRpcbPort), 10))
		if nil != err {
			return
		}

		udpConn, err = net.DialUDP("udp4", nil, udpAddr)
		if nil != err {
			return
		}

		bytesWritten, err = udpConn.Write(callBuf)
		if nil != err {
			_ = udpConn.Close()
			return
		}
		if len(callBuf) != bytesWritten {
			_ = udpConn.Close()
			err = fmt.Errorf("unable to write callBuf to UDP socket")
			return
		}

		replyBuf = make([]byte, onc.MaxUDPPacketSize, onc.MaxUDPPacketSize)

		bytesRead, err = udpConn.Read(replyBuf)
		if nil != err {
			_ = udpConn.Close()
			return
		}
		if 0 == bytesRead {
			_ = udpConn.Close()
			err = fmt.Errorf("unable to read replyBuf from UDP socket")
			return
		}

		replyBuf = replyBuf[:bytesRead]

		err = udpConn.Close()
		if nil != err {
			return
		}
	default:
		err = fmt.Errorf("prot == %v not supported", prot)
		return
	}

	port = onc.PmapAndRpcbPort // TODO: Actually need to process replyBuf to get it

	err = nil
	return
}
