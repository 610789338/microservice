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
	RegistRpcHandler	(name string, rpc RpcHandler)
	RpcParse			(buf []byte) uint32
	RpcDecode			(b []byte)
	RpcEncode			(name string, args ...interface{}) []byte
}

var rpcMgr RpcMgr = CreateSimpleRpcMgr()

func RegistRpcHandler(name string, rpc RpcHandler) {
	rpcMgr.RegistRpcHandler(name, rpc)
}

func Start(){
	runtime.GOMAXPROCS(runtime.NumCPU())
	INFO_LOG("ms start ...")

	netServer := CreateTcpServer("127.0.0.1", 6666)
	netServer.Start()

	INFO_LOG("ms stop ...")
}

