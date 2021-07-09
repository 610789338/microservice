package main


import (
    msf "ms_framework"
)

var gGateAddrMap = make(map[string]msf.CONN_ID)

func onTcpAccept(client *msf.TcpClient) {
    // 请求gate监听地址

    msf.DEBUG_LOG("on busi tcp accept %v", client.RemoteAddr())
    rpc := msf.RpcEncode(msf.MSG_P2G_REQ_LISTENADDR)
    msg := msf.MessageEncode(rpc)
    msf.MessageSend(client.GetConn(), msg)
}

func main() {
    
    msf.Init()

    msf.RegistRpcHandler(msf.MSG_PUSH_UNSAFE,             func() msf.RpcHandler {return new(RpcPushUnsafeHandler)})
    // msf.RegistRpcHandler(msf.MSG_PUSH_SAFE,             func() msf.RpcHandler {return new(RpcPushHandler)})
    msf.RegistRpcHandler(msf.MSG_G2P_RSP_LISTENADDR,     func() msf.RpcHandler {return new(RpcRspListenAddrHandler)})

    msf.SetBusOnTcpAccept(onTcpAccept)

    msf.Start()
}
