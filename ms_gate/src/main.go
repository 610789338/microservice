package main


import (
    msf "ms_framework"
    "fmt"
)

var gClientAccess = make(map[string]map[string]bool)

func main() {

    // gate分为内网和外网两种，内网gate承载内网的客户端（game server）连接
    if "ServiceGate" == msf.GlobalCfg.Service {
        msf.SetServerIdentity(msf.SERVER_IDENTITY_SERVICE_GATE)
    } else if "ClientGate" == msf.GlobalCfg.Service {
        msf.SetServerIdentity(msf.SERVER_IDENTITY_CLIENT_GATE)
    } else {
        panic(fmt.Sprintf("error service cfg %s", msf.GlobalCfg.Service))
    }

    msf.Init()
    
    msf.RegistRpcHandler(msf.MSG_C2G_RPC_ROUTE,           func() msf.RpcHandler {return new(RpcC2GRpcRouteHandler)})
    msf.RegistRpcHandler(msf.MSG_S2G_RPC_RSP,             func() msf.RpcHandler {return new(RpcS2GRpcRspHandler)})
    msf.RegistRpcHandler(msf.MSG_GATE_LOGIN,              func() msf.RpcHandler {return new(RpcGateLoginHandler)})
    msf.RegistRpcHandler(msf.MSG_GATE_LOGOFF,             func() msf.RpcHandler {return new(RpcGateLogoffHandler)})

    msf.RegistRpcHandler(msf.MSG_P2G_REQ_LISTENADDR,      func() msf.RpcHandler {return new(RpcReqListenAddrHandler)})
    msf.RegistRpcHandler(msf.MSG_P2G_PUSH,                func() msf.RpcHandler {return new(RpcP2GPushHandler)})
    msf.RegistRpcHandler(msf.MSG_C2G_VERTIFY,             func() msf.RpcHandler {return new(RpcC2GVertifyHandler)})
    msf.RegistRpcHandler(msf.MSG_S2G_RPC_ACCESS_REPORT,   func() msf.RpcHandler {return new(RpcS2GRpcAccessReportHandler)})

    msf.Start()
}
