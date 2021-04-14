package ms_framework

import (
	"runtime"
)


// func DEBUG_LOG(format string, params ...interface{} ) {msf_log.DEBUG_LOG(format, params...)}
// func INFO_LOG(format string, params ...interface{} ) {msf_log.INFO_LOG(format, params...)}
// func WARN_LOG(format string, params ...interface{} ) {msf_log.WARN_LOG(format, params...)}
// func ERROR_LOG(format string, params ...interface{} ) {msf_log.ERROR_LOG(format, params...)}

var USE_SIMPLE_RPC bool = true
var USE_TCP bool = true

type NetServer interface {
	Start()
	Close()
}

type RpcHandler interface {
	GetReqPtr() interface{}
	GetRspPtr() interface{}
	Process()
}

type RpcMgr interface {
	RegistRpcHandler	(name string, gen RpcHanderGenerator)
	MessageDecode		(buf []byte) (uint32, []byte)
	MessageEncode		(b []byte) []byte
	RpcDecode			(buf []byte) []byte
	RpcEncode			(name string, args ...interface{}) []byte
}

var rpcMgr RpcMgr = nil
var netServer *TcpServer = nil
var remoteMgr *RemoteMgr = nil

func RegistRpcHandler(name string, gen RpcHanderGenerator) {
	rpcMgr.RegistRpcHandler(name, gen)
}

func OnRemoteDiscover(namespace string, svrName string, ip string, port uint32) {
	remoteMgr.OnRemoteDiscover(namespace, svrName, ip, port)
}

func Init(ip string, port int) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	rpcMgr = CreateSimpleRpcMgr()
	rpcMgr.RegistRpcHandler(MSG_G2S_RPC_CALL, func() RpcHandler {return new(RpcG2SRpcCallHandler)})

	netServer = CreateTcpServer(ip, port)

	remoteMgr = CreateRemoteMgr()
	go remoteMgr.Start()

	INFO_LOG("ms init ok ...")
}

func Start() {
	INFO_LOG("ms start ...")
	netServer.Start()

	INFO_LOG("ms stop ...")
}
