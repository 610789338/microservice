// author: youjun
// date: 2021-03-11

package main


import (
	msf "ms_framework"
	"fmt"
)

func main() {
	msf.Init()

	if "ServiceGate" == msf.GlobalCfg.Service {
		msf.SetServerIdentity(msf.SERVER_IDENTITY_SERVICE_GATE)
	} else if "ClientGate" == msf.GlobalCfg.Service {
		msf.SetServerIdentity(msf.SERVER_IDENTITY_CLIENT_GATE)
	} else {
		panic(fmt.Sprint("error service cfg %s", msf.GlobalCfg.Service))
	}

	msf.Start()
}
