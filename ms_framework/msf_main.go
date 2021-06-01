package ms_framework

import (
	"runtime"
	"os/signal"
	"os"
	"syscall"
)

// 服务身份，分为微服务，微服务网关，客户端网关
// 微服务网关负责微服务以及游戏服务器之间的消息路由
// 客户端网关负责微服务和游戏客户端之间的消息路由
// 默认身份是微服务，网关服务需单独设置身份
const (
	SERVER_IDENTITY_SERVICE 	int8 = iota
	SERVER_IDENTITY_SERVICE_GATE
	SERVER_IDENTITY_CLIENT_GATE
)

var serverIdentity int8 = SERVER_IDENTITY_SERVICE // default

var IdentityMap = map[int8]string {
	SERVER_IDENTITY_SERVICE: "SERVER_IDENTITY_SERVICE",
	SERVER_IDENTITY_SERVICE_GATE: "SERVER_IDENTITY_SERVICE_GATE",
	SERVER_IDENTITY_CLIENT_GATE: "SERVER_IDENTITY_CLIENT_GATE",
}

func GetServerIdentity() int8 {
	return serverIdentity
}

func SetServerIdentity(identity int8) {
	serverIdentity = identity
}


// signal handler可重复注册
func SignalHander(handler func(), sig ...os.Signal) {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, sig...)
	<-sigChan
	handler()
}

var StartBusi func() = func() {}
var StopBusi func() = func() {}

func SetStartBusi(f func()) {
	StartBusi = f
}

func SetStopBusi(f func()) {
	StopBusi = f
}

func Init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	ParseArgs()
	
    SetLogLevel(GlobalCfg.LogLevel)

    if "ServiceGate" == GlobalCfg.Service {
        SetServerIdentity(SERVER_IDENTITY_SERVICE_GATE)
    } else if "ClientGate" == GlobalCfg.Service {
        SetServerIdentity(SERVER_IDENTITY_CLIENT_GATE)
    }

	CreateSimpleRpcMgr()
	CreateRemoteMgr()

	CreateTcpServer(GetTcpListenIP(), GlobalCfg.Port)
	CreateEtcdDriver()

	INFO_LOG("%s:%s init ok ...", GlobalCfg.Namespace, GlobalCfg.Service)
}

func Start() {
	INFO_LOG("%s:%s start ...", GlobalCfg.Namespace, GlobalCfg.Service)

	StartTaskPool()
	StartTcpServer()
	StartEtcdDriver()
	StartRpcFvc()
	StartBusi()

	// go SignalHander(Stop, syscall.SIGINT, syscall.SIGTERM),
	
	exitChan := make(chan os.Signal)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)
	<-exitChan

	Stop()

	INFO_LOG("%s:%s shutdown ...", GlobalCfg.Namespace, GlobalCfg.Service)
}

func Stop() {
	StopBusi()
	StopRpcFvc()
	StopEtcdDriver()
	StopTcpServer()
	StopTaskPool()
}
