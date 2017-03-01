package oncserver

import (
	"bytes"
	"net"
	"sync"
	"testing"

	"github.com/swiftstack/onc"
	"github.com/swiftstack/onc/oncclient"
	"github.com/swiftstack/xdr"
)

const (
	testAddr       = "127.0.0.1:51423"
	testPort       = uint16(51423)
	testProg       = uint32(0x40000000)
	testVers       = uint32(1)
	testProc       = uint32(1)
	testEchoString = "echo me"
)

type testProcArgsStruct struct {
	EchoString string `XDR_Name:"String"`
}

type testTCPRequestHandlerStruct struct {
	t *testing.T
}

type testUDPRequestHandlerStruct struct {
	t *testing.T
}

type testTCPReplyHandlerStruct struct {
	sync.WaitGroup
	t *testing.T
}

type testUDPReplyHandlerStruct struct {
	sync.WaitGroup
	t *testing.T
}

func TestAPI(t *testing.T) {
	var (
		err                   error
		progVersList          []ProgVersStruct
		tcpAddr               *net.TCPAddr
		tcpConn               *net.TCPConn
		testProcArgs          *testProcArgsStruct
		testTCPReplyHandler   *testTCPReplyHandlerStruct
		testTCPRequestHandler *testTCPRequestHandlerStruct
		testUDPReplyHandler   *testUDPReplyHandlerStruct
		testUDPRequestHandler *testUDPRequestHandlerStruct
		udpAddr               *net.UDPAddr
		udpConn               *net.UDPConn
	)

	testProcArgs = &testProcArgsStruct{EchoString: testEchoString}

	progVersList = []ProgVersStruct{ProgVersStruct{Prog: testProg, VersList: []uint32{testVers}}}

	testTCPRequestHandler = &testTCPRequestHandlerStruct{t: t}

	err = StartServer(onc.IPProtoTCP, testPort, progVersList, testTCPRequestHandler)
	if nil != err {
		t.Fatalf("StartServer(onc.IPProtoTCP, testPort, progVersList, testTCPRequestHandler) got error: %v", err)
	}

	tcpAddr, err = net.ResolveTCPAddr("tcp4", testAddr)
	if nil != err {
		t.Fatalf("net.ResolveTCPAddr(\"tcp4\", testAddr) got error: %v", err)
	}

	tcpConn, err = net.DialTCP("tcp4", nil, tcpAddr)
	if nil != err {
		t.Fatalf("net.DialTCP(\"tcp4\", nil, tcpAddr) got error: %v", err)
	}

	testTCPReplyHandler = &testTCPReplyHandlerStruct{t: t}

	testTCPReplyHandler.Add(1)

	err = oncclient.IssueTCPCall(tcpConn, testProg, testVers, testProc, nil, testProcArgs, testTCPReplyHandler)
	if nil != err {
		t.Fatalf("oncclient.IssueTCPCall(tcpConn, testProg, testVers, testProc, nil, testProcArgs, testTCPReplyHandler) got error: %v", err)
	}

	testTCPReplyHandler.Wait()

	err = tcpConn.Close()
	if nil != err {
		t.Fatalf("tcpConn.Close() got error: %v", err)
	}

	err = StopServer(onc.IPProtoTCP, testPort)
	if nil != err {
		t.Fatalf("StopServer(onc.IPProtoTCP, testPort) got error: %v", err)
	}

	testUDPRequestHandler = &testUDPRequestHandlerStruct{t: t}

	err = StartServer(onc.IPProtoUDP, testPort, progVersList, testUDPRequestHandler)
	if nil != err {
		t.Fatalf("StartServer(onc.IPProtoUDP, testPort, progVersList, testUDPRequestHandler) got error: %v", err)
	}

	udpAddr, err = net.ResolveUDPAddr("udp4", testAddr)
	if nil != err {
		t.Fatalf("net.ResolveUDPAddr(\"udp4\", testAddr) got error: %v", err)
	}

	udpConn, err = net.DialUDP("udp4", nil, udpAddr)
	if nil != err {
		t.Fatalf("net.DialUDP(\"udp4\", nil, udpAddr) got error: %v", err)
	}

	testUDPReplyHandler = &testUDPReplyHandlerStruct{t: t}

	testUDPReplyHandler.Add(1)

	err = oncclient.IssueUDPCall(udpConn, testProg, testVers, testProc, nil, testProcArgs, testUDPReplyHandler)
	if nil != err {
		t.Fatalf("oncclient.IssueUDPCall(udpConn, testProg, testVers, testProc, nil, testProcArgs, testUDPReplyHandler) got error: %v", err)
	}

	testUDPReplyHandler.Wait()

	err = udpConn.Close()
	if nil != err {
		t.Fatalf("udpConn.Close() got error: %v", err)
	}

	err = StopServer(onc.IPProtoUDP, testPort)
	if nil != err {
		t.Fatalf("StopServer(onc.IPProtoUDP, testPort) got error: %v", err)
	}
}

func (testTCPRequestHandler *testTCPRequestHandlerStruct) ONCRequest(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, authSysBody *onc.AuthSysBodyStruct, parms []byte) {
	var (
		bytesConsumed uint64
		err           error
		testProcArgs  testProcArgsStruct
	)

	bytesConsumed, err = xdr.Unpack(parms, &testProcArgs)
	if nil != err {
		testTCPRequestHandler.t.Fatalf("func (testTCPRequestHandler *testTCPRequestHandlerStruct) ONCRequest() call to xdr.Unpack(parms, &testProcArgs) got error: %v", err)
	}
	if uint64(len(parms)) != bytesConsumed {
		testTCPRequestHandler.t.Fatalf("func (testTCPRequestHandler *testTCPRequestHandlerStruct) ONCRequest() call to xdr.Unpack(parms, &testProcArgs) didn't consume entire parms")
	}
	if testEchoString != testProcArgs.EchoString {
		testTCPRequestHandler.t.Fatalf("func (testTCPRequestHandler *testTCPRequestHandlerStruct) ONCRequest() testEchoString != testProcArgs.echoString")
	}

	err = SendAcceptedSuccess(connHandle, xid, []byte(testEchoString))
	if nil != err {
		testTCPRequestHandler.t.Fatalf("func (testTCPRequestHandler *testTCPRequestHandlerStruct) ONCRequest() call to SendAcceptedSuccess() got error: %v", err)
	}
}

func (testUDPRequestHandler *testUDPRequestHandlerStruct) ONCRequest(connHandle ConnHandle, xid uint32, prog uint32, vers uint32, proc uint32, authSysBody *onc.AuthSysBodyStruct, parms []byte) {
	var (
		bytesConsumed uint64
		err           error
		testProcArgs  testProcArgsStruct
	)

	bytesConsumed, err = xdr.Unpack(parms, &testProcArgs)
	if nil != err {
		testUDPRequestHandler.t.Fatalf("func (testUDPRequestHandler *testUDPRequestHandlerStruct) ONCRequest() call to xdr.Unpack(parms, &testProcArgs) got error: %v", err)
	}
	if uint64(len(parms)) != bytesConsumed {
		testUDPRequestHandler.t.Fatalf("func (testUDPRequestHandler *testUDPRequestHandlerStruct) ONCRequest() call to xdr.Unpack(parms, &testProcArgs) didn't consume entire parms")
	}
	if testEchoString != testProcArgs.EchoString {
		testUDPRequestHandler.t.Fatalf("func (testUDPRequestHandler *testUDPRequestHandlerStruct) ONCRequest() testEchoString != testProcArgs.echoString")
	}

	err = SendAcceptedSuccess(connHandle, xid, []byte(testEchoString))
	if nil != err {
		testUDPRequestHandler.t.Fatalf("func (testUDPRequestHandler *testUDPRequestHandlerStruct) ONCRequest() call to SendAcceptedSuccess() got error: %v", err)
	}
}

func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplySuccess(results []byte) {
	if 0 != bytes.Compare([]byte(testEchoString), results) {
		testTCPReplyHandler.t.Fatalf("func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplySuccess() received unexpected parms")
	}

	testTCPReplyHandler.Done()
}

func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplyProgMismatch(mismatchInfoLow uint32, mismatchInfoHigh uint32) {
	testTCPReplyHandler.t.Fatalf("func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplyProgMismatch() unexpectedly called")
}

func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplyRpcMismatch(mismatchInfoLow uint32, mismatchInfoHigh uint32) {
	testTCPReplyHandler.t.Fatalf("func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplyRpcMismatch() unexpectedly called")
}

func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplyAuthError(stat uint32) {
	testTCPReplyHandler.t.Fatalf("func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplyAuthError() unexpectedly called")
}

func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplyConnectionDown() {
	testTCPReplyHandler.t.Fatalf("func (testTCPReplyHandler *testTCPReplyHandlerStruct) ONCReplyConnectionDown() unexpectedly called")
}

func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplySuccess(results []byte) {
	if 0 != bytes.Compare([]byte(testEchoString), results) {
		testUDPReplyHandler.t.Fatalf("func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplySuccess() received unexpected parms")
	}

	testUDPReplyHandler.Done()
}

func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplyProgMismatch(mismatchInfoLow uint32, mismatchInfoHigh uint32) {
	testUDPReplyHandler.t.Fatalf("func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplyProgMismatch() unexpectedly called")
}

func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplyRpcMismatch(mismatchInfoLow uint32, mismatchInfoHigh uint32) {
	testUDPReplyHandler.t.Fatalf("func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplyRpcMismatch() unexpectedly called")
}

func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplyAuthError(stat uint32) {
	testUDPReplyHandler.t.Fatalf("func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplyAuthError() unexpectedly called")
}

func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplyConnectionDown() {
	testUDPReplyHandler.t.Fatalf("func (testUDPReplyHandler *testUDPReplyHandlerStruct) ONCReplyConnectionDown() unexpectedly called")
}
