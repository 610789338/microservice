// author: youjun
// date: 2021-03-11

package main


import (
    msf "ms_framework"
)

func onServiceStart() {
    msf.INFO_LOG("%s business start", msf.GlobalCfg.Service)
}

func onServiceStop() {
    msf.INFO_LOG("%s business stop", msf.GlobalCfg.Service)
}

func main(){
    msf.Init()
    msf.SetStartBusi(onServiceStart)
    msf.SetStopBusi(onServiceStop)

    msf.RegistRpcHandler("rpc_b", func() msf.RpcHandler {return new(RpcBHandler)})

    msf.Start()
}
