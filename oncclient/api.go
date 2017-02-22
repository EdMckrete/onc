package oncclient

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/swiftstack/onc"
	"github.com/swiftstack/xdr"
)

type globalsStruct struct {
	sync.Mutex
	nextXID uint32
}

var globals globalsStruct

// FetchXID fetches the next available XID
func FetchXID() (xid uint32) {
	globals.Lock()
	xid = globals.nextXID
	globals.nextXID++
	globals.Unlock()
	return
}

// LocateService returns the port number serving the specified program/version/protocol.
func LocateService(addr string, prog uint32, vers uint32, prot uint32) (port uint32, err error) {
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

	xid = FetchXID()

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

		replyRecordFragmentMarkBuf = make([]byte, 0, replyRecordFragmentMarkBufLen)

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

		replyBuf = make([]byte, 0, (replyRecordFragmentMark.LastFragmentFlagAndFragmentLength & 0x7FFFFFFF))

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

		replyBuf = make([]byte, 0, onc.MaxUDPPacketSize)

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

		err = udpConn.Close()
		if nil != err {
			return
		}
	default:
		err = fmt.Errorf("prot == %v not supported", prot)
		return
	}

	err = nil
	return
}
