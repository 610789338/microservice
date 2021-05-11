package ms_framework

import (
	"runtime"
	"os/signal"
	"os"
	"syscall"
)


// var USE_SIMPLE_RPC bool = true
// var USE_TCP bool = true

// type NetServer interface {
// 	Start()
// 	Close()
// }

// type RpcMgr interface {
// 	RegistRpcHandler		(name string, gen RpcHanderGenerator)
// 	MessageDecode			(buf []byte) (uint32, []byte)
// 	MessageEncode			(b []byte) []byte
// 	RpcDecode				(buf []byte) []byte
// 	RpcEncode				(name string, args ...interface{}) []byte
// 	GetRpcHanderGenerator	(rpcName string) (RpcHanderGenerator, bool)
// }

// 可重复注册signal handler
func SignalHander(handler func(), sig ...os.Signal) {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, sig...)
	<-sigChan
	handler()
}

var BusiStop func()

func SetBusiStop(f func()) {
	BusiStop = f
}

func Init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	ParseArgs()

	CreateSimpleRpcMgr()
	CreateRemoteMgr()

	CreateTcpServer("", GlobalCfg.Port)
	CreateEtcdDriver()

	INFO_LOG("%s:%s init ok ...", GlobalCfg.Namespace, GlobalCfg.Service)
}

func Start() {
	INFO_LOG("%s:%s start ...", GlobalCfg.Namespace, GlobalCfg.Service)

	StartTcpServer()
	StartEtcdDriver()

	// go SignalHander(Stop, syscall.SIGINT, syscall.SIGTERM)
	
	exitChan := make(chan os.Signal)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)
	<-exitChan

	Stop()

	INFO_LOG("%s:%s shutdown ...", GlobalCfg.Namespace, GlobalCfg.Service)
}

func Stop() {
	if BusiStop != nil {
		BusiStop()
	}

	StopEtcdDriver()
	StopTcpServer()
}
