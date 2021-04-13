// author: youjun
// date: 2021-03-11

package main


import (
	msf "ms_framework"
)


func main() {
	
	msf.Init("127.0.0.1", 8888)
	msf.RegistRpcHandler(msf.MSG_C2G_RPC_ROUTE, 	func() msf.RpcHandler {return new(RpcC2GRpcRouteHandler)})
	msf.RegistRpcHandler(msf.MSG_COMMON_RSP, 		func() msf.RpcHandler {return new(RpcS2GCommonRspHandler)})

	msf.OnRemoteDiscover("", "testService", "127.0.0.1", 6666)

	msf.Start()
}
