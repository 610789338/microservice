// author: youjun
// date: 2021-03-11

package main


import (
	msf "ms_framework"
	"fmt"
)

func main() {
	msf.Init()

	if msf.GlobalCfg.Service != "ServiceGate" && msf.GlobalCfg.Service != "ClientGate" {
		panic(fmt.Sprintf("error service cfg %s", msf.GlobalCfg.Service))
	}

	msf.Start()
}
