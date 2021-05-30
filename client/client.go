package main

import (
	msf "ms_framework"
	"time"
	"clientsdk"
	"math/rand"
	"sync"
)

var namespace = "YJ"

var doMutex sync.Mutex

var gate *clientsdk.GateProxy

func init() {
	clientsdk.Init()
	gate = clientsdk.CreateGateProxy("127.0.0.1", 8886)
	if nil == gate {
		panic("gate is nil~~~")
	}
}

func main() {
	msf.INFO_LOG("clientsdk start %v", time.Now())
	TestService := gate.CreateServiceProxy(namespace, "ServiceA")
	methodName := "rpc_a"
	// methodName := "rpc_b"

	startTs := time.Now().UnixNano() / 1e6
	time.Sleep(time.Millisecond)
	var total, do, i int32 = 10000, 0, 0
	for ; i < total; i++ {
		TestService.RpcCall(methodName, i, rand.Float32(), "abc", map[string]interface{}{"key1": rand.Int63(), "key2": "def"}, []int32{rand.Int31(), rand.Int31()}, 
		// TestService.RpcCall(methodName, i, 
		clientsdk.CallBack(func(err string, reply map[string]interface{}) {
			if err != "" {
				msf.ERROR_LOG("[rpc call] - %s response: err(%v) reply(%v)", methodName, err, reply)
				return
			}

			// msf.INFO_LOG("[%s] response: err(%v) reply(%v)", methodName, err, reply)
			// var req int32
			// switch reply["Req"].(type){
			// case int8:
			// 	req = int32(reply["Req"].(int8))
			// case uint8:
			// 	req = int32(reply["Req"].(uint8))
			// case int16:
			// 	req = int32(reply["Req"].(int16))
			// case uint16:
			// 	req = int32(reply["Req"].(uint16))
			// case int32:
			// 	req = int32(reply["Req"].(int32))
			// case uint32:
			// 	req = int32(reply["Req"].(uint32))
			// }

			doMutex.Lock()
			do += 1
			doMutex.Unlock()
		}))

		// time.Sleep(time.Millisecond)
	}

	endTs := time.Now().UnixNano() / 1e6
	ops := int64(total*1000)/(endTs - startTs)
	msf.DEBUG_LOG("send: startTs %v, endTs %v, ops = %v", startTs, endTs, ops)

	for true {
		if do >= total {
			break
		}
		time.Sleep(time.Millisecond)
	}

	endTs = time.Now().UnixNano() / 1e6

	ops = int64(total*1000)/(endTs - startTs)
	msf.DEBUG_LOG("rtt: startTs %v, endTs %v, ops = %v", startTs, endTs, ops)
}
