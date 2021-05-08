// author: youjun
// date: 2021-03-11

package main


import (
	msf "ms_framework"
)

func onServiceStop() {
	msf.INFO_LOG("%s business stop", msf.GlobalCfg.Service)
}

func main(){
	msf.Init()

	msf.RegistRpcHandler("rpc_test", func() msf.RpcHandler {return new(RpcTestHandler)})
	msf.RegistRpcHandler("rpc_test1", func() msf.RpcHandler {return new(RpcTest1Handler)})

	msf.SetBusiStop(onServiceStop)

	msf.Start()
}
