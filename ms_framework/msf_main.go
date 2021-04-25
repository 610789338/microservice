package ms_framework

import (
	"runtime"
)


var USE_SIMPLE_RPC bool = true
var USE_TCP bool = true

type NetServer interface {
	Start()
	Close()
}

// type RpcMgr interface {
// 	RegistRpcHandler		(name string, gen RpcHanderGenerator)
// 	MessageDecode			(buf []byte) (uint32, []byte)
// 	MessageEncode			(b []byte) []byte
// 	RpcDecode				(buf []byte) []byte
// 	RpcEncode				(name string, args ...interface{}) []byte
// 	GetRpcHanderGenerator	(rpcName string) (RpcHanderGenerator, bool)
// }


func OnRemoteDiscover(namespace string, svrName string, ip string, port uint32) {
	remoteMgr.OnRemoteDiscover(namespace, svrName, ip, port)
}

func Init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	CreateSimpleRpcMgr()
	CreateTcpServer("", GlobalCfg.Port)
	CreateRemoteMgr()

	INFO_LOG("ms init ok ...")
}

func Start() {
	INFO_LOG("ms start ...")
	TcpServerStart()
	INFO_LOG("ms stop ...")
}
