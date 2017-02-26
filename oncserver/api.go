package oncserver

import (
	"github.com/swiftstack/onc"
)

// ProgVersStruct represents an ONC RPC Programs and the list of supported Versions.
type ProgVersStruct struct {
	Prog     uint32
	VersList []uint32
}

type RequestCallbacks interface {
	ONCRequest(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, authSysBody *onc.AuthSysBodyStruct, parms []byte)
}

// StartServer initiates a server on the specified protocol/port for the listed ONC RPC Programs/Versions.
//
// The caller may optionally request that the service(s) be registered with the local portmapper/rpcbind instance.
func StartServer(prot uint32, port uint16, progVersList []ProgVersStruct, register bool, callbacks RequestCallbacks) (err error) {
	err = startServer(prot, port, progVersList, register, callbacks)
	return
}

// StopServer halts serving of the ONC RPC Programs/Versions on the specified protocol/port.
//
// The caller may optionally request that the services() be deregistered from the local portmapper/rpcbind instance.
func StopServer(prot uint32, port uint16, deregister bool) (err error) {
	err = stopServer(prot, port, deregister)
	return
}

// SendDeniedRpcMismatchReply is used to send a MsgDenied-RpcMismatch reply for an ONC RPC reqest.
func SendDeniedRpcMismatchReply(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, mismatchInfoLow uint32, mismatchInfoHigh uint32) (err error) {
	err = sendDeniedRpcMismatchReply(prot, port, xid, prog, vers, proc, mismatchInfoLow, mismatchInfoHigh)
	return
}

// SendDeniedAuthErrorReply is used to send a MsgDenied-AuthError reply for an ONC RPC request.
func SendDeniedAuthErrorReply(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, stat uint32) (err error) {
	err = sendDeniedAuthErrorReply(prot, port, xid, prog, vers, proc, stat)
	return
}

// SendAcceptedSuccess is used to send a Success reply for an ONC RPC request.
func SendAcceptedSuccess(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, results []byte) (err error) {
	err = sendAcceptedSuccess(prot, port, xid, prog, vers, proc, results)
	return
}

// SendAcceptedProgMismatchReply is used to send a ProgMismatch reply for an ONC RPC request.
func SendAcceptedProgMismatchReply(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, mismatchInfoLow uint32, mismatchInfoHigh uint32) (err error) {
	err = sendAcceptedProgMismatchReply(prot, port, xid, prog, vers, proc, mismatchInfoLow, mismatchInfoHigh)
	return
}

// SendAcceptedOtherErrorReply is used to send a ProgUnavail, ProcUnavail, GarbageArgs, or SystemErr reply for an ONC RPC request.
func SendAcceptedOtherErrorReply(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, stat uint32) (err error) {
	err = sendAcceptedOtherErrorReply(prot, port, xid, prog, vers, proc, stat)
	return
}
