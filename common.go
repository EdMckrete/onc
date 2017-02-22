package onc

const ( // enum auth_flavor
	AuthNone  = uint32(0)
	AuthSys   = uint32(1)
	AuthShort = uint32(2)
	AuthDH    = uint32(3)
	AuthGSS   = uint32(6)
)

type OpaqueAuthStruct struct {
	AuthFlavor uint32 `XDR_Name:"Enumeration"` // One of enum auth_flavor
	OpaqueBody []byte `XDR_Name:"Variable-Length Opaque Data" XDR_MaxSize:"400"`
}

type AuthSysBodyStruct struct {
	Stamp       uint32   `XDR_Name:"Unsigned Integer"`
	MachineName string   `XDR_Name:"String" XDR_MaxSize:"255"`
	UID         uint32   `XDR_Name:"Unsigned Integer"`
	GID         uint32   `XDR_Name:"Unsigned Integer"`
	GIDs        []uint32 `XDR_Name:"Variable-Length Array" XDR_MaxSize:"16"`
}

const ( // Program Number Assignment
	ProgNumPortMap = uint32(100000)
	ProgNumNFS     = uint32(100003)
	ProgNumMount   = uint32(100005)
	//                              Note: There are many more... but we are only trying to support NFS for now
)

const ( // enum msg_type
	Call  = uint32(0)
	Reply = uint32(1)
)

const ( // enum reply_stat
	MsgAccepted = uint32(0)
	MsgDenied   = uint32(1)
)

const ( // enum accept_stat
	Success      = uint32(0)
	ProgUnavail  = uint32(1)
	ProgMismatch = uint32(2)
	ProcUnavail  = uint32(3)
	GarbageArgs  = uint32(4)
	SystemErr    = uint32(5)
)

const ( // enum reject_stat
	RpcMismatch = uint32(0)
	AuthError   = uint32(1)
)

const ( // enum auth_stat
	AuthOk               = uint32(0)
	AuthBadCred          = uint32(1)
	AuthRejectedCred     = uint32(2)
	AuthBadVerf          = uint32(3)
	AuthRejectedVerf     = uint32(4)
	AuthTooWeak          = uint32(5)
	AuthInvalidResp      = uint32(6)
	AuthFailed           = uint32(7)
	AuthKerbGeneric      = uint32(8)
	AuthTimeExpire       = uint32(9)
	AuthTktFile          = uint32(10)
	AuthDecode           = uint32(11)
	AuthNetAddr          = uint32(12)
	RpcSecGssCredProblem = uint32(13)
	RpcSecGssCtxProblem  = uint32(14)
)

type RpcMsgHeaderStruct struct {
	XID   uint32 `XDR_Name:"Unsigned Integer"`
	MType uint32 `XDR_Name:"Enumeration"` //   One of enum msg_type
	//                                         If MType == Call,  RpcMsgCallBodyHeaderStruct  starts here
	//                                         If MType == Reply, RpcMsgReplyBodyHeaderStruct starts here
}

const (
	RpcVers2 = uint32(2) // Only supported ONC RPC Version
)

type RpcMsgCallBodyHeaderStruct struct {
	RpcVers uint32           `XDR_Name:"Unsigned Integer"` // Must be RpcVers2
	Prog    uint32           `XDR_Name:"Unsigned Integer"` // One of above listed Program Number Assignment values
	Vers    uint32           `XDR_Name:"Unsigned Integer"`
	Proc    uint32           `XDR_Name:"Unsigned Integer"`
	Cred    OpaqueAuthStruct `XDR_Name:"Structure"`
	Verf    OpaqueAuthStruct `XDR_Name:"Structure"`
	//                                                        Procedure-specific parameters start here
}

type RpcMsgReplyBodyHeaderStruct struct {
	Stat uint32 `XDR_Name:"Enumeration"` // One of enum reply_stat
	//                                      If Stat == MsgAccepted, RpcMsgReplyBodyAcceptedHeaderStruct starts here
	//                                      If Stat == MsgDenied,   RpcMsgReplyBodyRejectedHeaderStruct starts here
}

type RpcMsgReplyBodyAcceptedHeaderStruct struct {
	Verf OpaqueAuthStruct `XDR_Name:"Structure"`
	Stat uint32           `XDR_Name:"Enumeration"` // One of enum accept_stat
	//                                                If Stat == Success,      procedure-specific results                start  here
	//                                                If Stat == ProgMismatch, RpcMsgReplyBodyAcceptedProgMismatchStruct starts here
}

type RpcMsgReplyBodyAcceptedProgMismatchStruct struct {
	MismatchInfoLow  uint32 `XDR_Name:"Unsigned Integer"`
	MismatchInfoHigh uint32 `XDR_Name:"Unsigned Integer"`
}

type RpcMsgReplyBodyRejectedHeaderStruct struct {
	Stat uint32 `XDR_Name:"Enumeration"` // One of enum reject_stat
	//                                      If Stat == RpcMismatch, RpcMsgReplyBodyRejectedRpcMismatchStruct starts here
	//                                      If Stat == AuthError,   RpcMsgReplyBodyRejectedAuthErrorStruct   starts here
}

type RpcMsgReplyBodyRejectedRpcMismatchStruct struct {
	MismatchInfoLow  uint32 `XDR_Name:"Unsigned Integer"`
	MismatchInfoHigh uint32 `XDR_Name:"Unsigned Integer"`
}

type RpcMsgReplyBodyRejectedAuthErrorStruct struct {
	Stat uint32 `XDR_Name:"Enumeration"` // One of enum auth_stat
}

type RecordFragmentMarkStruct struct { //                                     Used for TCP
	LastFragmentFlagAndFragmentLength uint32 `XDR_Name:"Unsigned Integer"` // BigEndian with high-order bit indicating:
	//                                                                          0: not last fragment
	//                                                                          1: last fragment
	//                                                                        Remaining 31 bits indicating fragment length
	//                                                                        Note: Currently, only a single fragment record is supported
}

const (
	PmapAndRpcbPort = uint32(111) // Used for Portmapper (Program Version 2) & Rpcbind (Program Versions 3 & 4)
	//                               Note 1: Must actually fit in a uint16
	//                               Note 2: Portmapper/Rpcbind typically listens on both UDP and TCP
)

const (
	PmapVers  = uint32(2) // Portmapper protocol (Program Version 2)
	RpcbVers  = uint32(3) // Rpcbind protocol    (Program Version 3)
	RpcbVers4 = uint32(4) // Rpcbind protocol    (Program Version 4)
)

const ( // Used in the Prot field of MappingStruct below
	IPProtoTCP = uint32(6)
	IPProtoUDP = uint32(17)
)

const ( // enum pmap_proc
	PmapProcNull    = uint32(0)
	PmapProcSet     = uint32(1)
	PmapProcUnset   = uint32(2)
	PmapProcGetAddr = uint32(3)
	//                          Note: There are many more... but we only need these for now
)

type PmapMappingStruct struct { // Procedure-specific parameters for PmapProcSet, PmapProcUnset, & PmapProcGetAddr
	Prog uint32 `XDR_Name:"Unsigned Integer"` // One of above listed Program Number Assignment values
	Vers uint32 `XDR_Name:"Unsigned Integer"`
	Prot uint32 `XDR_Name:"Unsigned Integer"` // [Ignored for PmapProcUnset]                   One of IPProtoTCP or IPProtoUDP
	Port uint32 `XDR_Name:"Unsigned Integer"` // [Ignored for PmapProcUnset & PmapProcGetAddr] Note: Must actually fit in a uint16
}

type PmapMappingResponseStruct struct { // Procedure-specific results for PmapProcSet & PmapProcUnset
	Success bool `XDR_Name:"Boolean"`
}

type PmapGetAddrResponseStruct struct { // Procedure-specific results for PmapProcGetAddr
	Port uint32 `XDR_Name:"Unsigned Integer"` // Note 1: Must actually fit in a uint16
	//                                           Note 2: A value of Zero means the Program is not registered
}

const ( // enum Rpcb_proc
	RpcbProcSet     = uint32(1)
	RpcbProcUnset   = uint32(2)
	RpcbProcGetAddr = uint32(3)
	//                          Note: There are many more... but we only need these for now
)

const ( // values for NetID
	NcTCP  = "tcp"  // TCP over IPv4
	NcTCP6 = "tcp6" // TCP over IPv6
	NcUDP  = "udp"  // UDP over IPv4
	NcUDP6 = "udp6" // UDP over IPv6
	//                 Note: There are many more... but we only need these for now
)

type RpcbStruct struct { // Procedure-specific parameters for RpcbProcSet & RpcbProcUnset
	Prog  uint32 `XDR_Name:"Unsigned Integer"` // One of above listed Program Number Assignment values
	Vers  uint32 `XDR_Name:"Unsigned Integer"`
	NetId string `XDR_Name:"String"` //           One of NcTCP, NcTCP6, NcUDP, or NcUDP6
	Addr  string `XDR_Name:"String"` //           For IPv4: h1.h2.h3.h4.p1.p2
	//                                            For IPv6: x1:x2:x3:x4:x5:x6:x7:x8.p1.p2
	//                                            Note: Only IPv4 will be supported for now
	Owner string `XDR_Name:"String"`
}

type RpcbMappingResponseStruct struct { // Procedure-specific results for RpcbProcSet & RpcbProcUnset
	Success bool `XDR_Name:"Boolean"`
}

type RpcbGetAddrResponseStruct struct { // Procedure-specific results for RpcbProcGetAddr
	Addr string `XDR_Name:"String"` // For IPv4: h1.h2.h3.h4.p1.p2
	//                                 For IPv6: x1:x2:x3:x4:x5:x6:x7:x8.p1.p2
	//                                 Note: Only IPv4 will be supported for now
}

const (
	MaxUDPPacketSize = int(0x10000)
)
