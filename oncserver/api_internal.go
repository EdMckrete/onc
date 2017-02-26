package oncserver

func startServer(prot uint32, port uint16, progVersList []ProgVersStruct, register bool, callbacks RequestCallbacks) (err error) {
	err = nil // TODO
	return
}

func stopServer(prot uint32, port uint16, deregister bool) (err error) {
	err = nil // TODO
	return
}

func sendDeniedRpcMismatchReply(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, mismatchInfoLow uint32, mismatchInfoHigh uint32) (err error) {
	err = nil // TODO
	return
}

func sendDeniedAuthErrorReply(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, stat uint32) (err error) {
	err = nil // TODO
	return
}

func sendAcceptedSuccess(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, results []byte) (err error) {
	err = nil // TODO
	return
}

func sendAcceptedProgMismatchReply(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, mismatchInfoLow uint32, mismatchInfoHigh uint32) (err error) {
	err = nil // TODO
	return
}

func sendAcceptedOtherErrorReply(prot uint32, port uint16, xid uint32, prog uint32, vers uint32, proc uint32, stat uint32) (err error) {
	err = nil // TODO
	return
}
