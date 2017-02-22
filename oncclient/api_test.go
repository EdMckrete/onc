package oncclient

import (
	"fmt"
	"testing"

	"github.com/swiftstack/onc"
)

const (
	testExpectedMountPort = uint16(1048)
	testExpectedNFSPort   = uint16(2049)
)

func TestAPI(t *testing.T) {
	var (
		err                 error
		mountPort           uint16
		nfsPort             uint16
		portmapperV2TCPPort uint16
		portmapperV2UDPPort uint16
		rpcbindV3TCPPort    uint16
		rpcbindV3UDPPort    uint16
		rpcbindV4TCPPort    uint16
		rpcbindV4UDPPort    uint16
	)

	portmapperV2TCPPort, err = DoPmapProcGetAddr("127.0.0.1", onc.ProgNumPortMap, onc.PmapVers, onc.IPProtoTCP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != portmapperV2TCPPort {
		err = fmt.Errorf("portmapperV2TCPPort value unexpected (%v)... should have been %v", portmapperV2TCPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	portmapperV2UDPPort, err = DoPmapProcGetAddr("127.0.0.1", onc.ProgNumPortMap, onc.PmapVers, onc.IPProtoUDP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != portmapperV2UDPPort {
		err = fmt.Errorf("portmapperV2UDPPort value unexpected (%v)... should have been %v", portmapperV2UDPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	rpcbindV3TCPPort, err = DoPmapProcGetAddr("127.0.0.1", onc.ProgNumPortMap, onc.RpcbVers, onc.IPProtoTCP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != rpcbindV3TCPPort {
		err = fmt.Errorf("rpcbindV3TCPPort value unexpected (%v)... should have been %v", rpcbindV3TCPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	rpcbindV3UDPPort, err = DoPmapProcGetAddr("127.0.0.1", onc.ProgNumPortMap, onc.RpcbVers, onc.IPProtoUDP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != rpcbindV3UDPPort {
		err = fmt.Errorf("rpcbindV3UDPPort value unexpected (%v)... should have been %v", rpcbindV3UDPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	rpcbindV4TCPPort, err = DoPmapProcGetAddr("127.0.0.1", onc.ProgNumPortMap, onc.RpcbVers4, onc.IPProtoTCP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != rpcbindV4TCPPort {
		err = fmt.Errorf("rpcbindV4TCPPort value unexpected (%v)... should have been %v", rpcbindV4TCPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	rpcbindV4UDPPort, err = DoPmapProcGetAddr("127.0.0.1", onc.ProgNumPortMap, onc.RpcbVers4, onc.IPProtoUDP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != rpcbindV4UDPPort {
		err = fmt.Errorf("rpcbindV4UDPPort value unexpected (%v)... should have been %v", rpcbindV4UDPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	err = DoPmapProcSet(onc.ProgNumMount, 3, onc.IPProtoUDP, testExpectedMountPort)
	if nil != err {
		t.Fatal(err)
	}

	mountPort, err = DoPmapProcGetAddr("127.0.0.1", onc.ProgNumMount, 3, onc.IPProtoUDP)
	if nil != err {
		t.Fatal(err)
	}
	if testExpectedMountPort != mountPort {
		err = fmt.Errorf("mountPort value unexpected (%v)... should have been %v", mountPort, testExpectedMountPort)
		t.Fatal(err)
	}

	err = DoPmapProcUnset(onc.ProgNumMount, 3, onc.IPProtoUDP)
	if nil != err {
		t.Fatal(err)
	}

	err = DoPmapProcSet(onc.ProgNumNFS, 3, onc.IPProtoTCP, testExpectedNFSPort)
	if nil != err {
		t.Fatal(err)
	}

	nfsPort, err = DoPmapProcGetAddr("127.0.0.1", onc.ProgNumNFS, 3, onc.IPProtoTCP)
	if nil != err {
		t.Fatal(err)
	}
	if testExpectedNFSPort != nfsPort {
		err = fmt.Errorf("nfsPort value unexpected (%v)... should have been %v", nfsPort, testExpectedNFSPort)
		t.Fatal(err)
	}

	err = DoPmapProcUnset(onc.ProgNumNFS, 3, onc.IPProtoTCP)
	if nil != err {
		t.Fatal(err)
	}
}
