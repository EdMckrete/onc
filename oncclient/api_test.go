package oncclient

import (
	"fmt"
	"testing"

	"github.com/swiftstack/onc"
)

func TestAPI(t *testing.T) {
	var (
		err                 error
		portmapperV2TCPPort uint16
		portmapperV2UDPPort uint16
		rpcbindV3TCPPort    uint16
		rpcbindV3UDPPort    uint16
		rpcbindV4TCPPort    uint16
		rpcbindV4UDPPort    uint16
	)

	portmapperV2TCPPort, err = LocateService("127.0.0.1", onc.ProgNumPortMap, onc.PmapVers, onc.IPProtoTCP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != portmapperV2TCPPort {
		err = fmt.Errorf("portmapperV2TCPPort value unexpected (%v)... should have been %v", portmapperV2TCPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	portmapperV2UDPPort, err = LocateService("127.0.0.1", onc.ProgNumPortMap, onc.PmapVers, onc.IPProtoUDP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != portmapperV2UDPPort {
		err = fmt.Errorf("portmapperV2UDPPort value unexpected (%v)... should have been %v", portmapperV2UDPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	rpcbindV3TCPPort, err = LocateService("127.0.0.1", onc.ProgNumPortMap, onc.RpcbVers, onc.IPProtoTCP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != rpcbindV3TCPPort {
		err = fmt.Errorf("rpcbindV3TCPPort value unexpected (%v)... should have been %v", rpcbindV3TCPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	rpcbindV3UDPPort, err = LocateService("127.0.0.1", onc.ProgNumPortMap, onc.RpcbVers, onc.IPProtoUDP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != rpcbindV3UDPPort {
		err = fmt.Errorf("rpcbindV3UDPPort value unexpected (%v)... should have been %v", rpcbindV3UDPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	rpcbindV4TCPPort, err = LocateService("127.0.0.1", onc.ProgNumPortMap, onc.RpcbVers4, onc.IPProtoTCP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != rpcbindV4TCPPort {
		err = fmt.Errorf("rpcbindV4TCPPort value unexpected (%v)... should have been %v", rpcbindV4TCPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}

	rpcbindV4UDPPort, err = LocateService("127.0.0.1", onc.ProgNumPortMap, onc.RpcbVers4, onc.IPProtoUDP)
	if nil != err {
		t.Fatal(err)
	}
	if onc.PmapAndRpcbPort != rpcbindV4UDPPort {
		err = fmt.Errorf("rpcbindV4UDPPort value unexpected (%v)... should have been %v", rpcbindV4UDPPort, onc.PmapAndRpcbPort)
		t.Fatal(err)
	}
}
