package oncclient

import (
	"net"

	"github.com/swiftstack/onc"
)

type ReplyCallbacks interface {
	ONCReplySuccess(results []byte)
	ONCReplyProgMismatch(mismatchInfoLow uint32, mismatchInfoHigh uint32)
	ONCReplyRpcMismatch(mismatchInfoLow uint32, mismatchInfoHigh uint32)
	ONCReplyAuthError(stat uint32)
	ONCReplyConnectionDown()
}

// IssueTCPCall sends the ONC RPC Call on the supplied *net.TCPConn
func IssueTCPCall(tcpConn *net.TCPConn, prog uint32, vers uint32, proc uint32, authSysBody *onc.AuthSysBodyStruct, procArgs interface{}, replyCallbacks ReplyCallbacks) (err error) {
	err = issueTCPCall(tcpConn, prog, vers, proc, authSysBody, procArgs, replyCallbacks)
	return
}

// IssueUDPCall sends the ONC RPC Call on the supplied *net.UDPConn
func IssueUDPCall(udpConn *net.UDPConn, prog uint32, vers uint32, proc uint32, authSysBody *onc.AuthSysBodyStruct, procArgs interface{}, replyCallbacks ReplyCallbacks) (err error) {
	err = issueUDPCall(udpConn, prog, vers, proc, authSysBody, procArgs, replyCallbacks)
	return
}

// DoPmapProcSet sets the portmapper record for the specified program/version/protocol
func DoPmapProcSet(prog uint32, vers uint32, prot uint32, port uint16) (err error) {
	err = doPmapProcSet(prog, vers, prot, port)
	return nil
}

// DoPmapProcUnset un-sets the portmapper record for the specified program/version/protocol
func DoPmapProcUnset(prog uint32, vers uint32, prot uint32) (err error) {
	err = doPmapProcUnset(prog, vers, prot)
	return nil
}

// DoPmapProcGetAddr returns the port number serving the specified program/version/protocol
func DoPmapProcGetAddr(addr string, prog uint32, vers uint32, prot uint32) (port uint16, err error) {
	port, err = doPmapProcGetAddr(addr, prog, vers, prot)
	return
}
