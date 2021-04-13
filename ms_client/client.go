package main

import (
	msf "ms_framework"
	"time"
	"clientsdk"
)

func main() {
	clientsdk.Init()
	gate := clientsdk.CreateGateProxy("127.0.0.1", 8888)
	TestService := gate.CreateServiceProxy("", "testService")
	methodName := "rpc_test"
	for true {
		TestService.RpcCall(methodName, 10, float32(9.9), "abc", map[string]interface{}{"key1": 10, "key2": "def"}, []int32{123, 456}, 
		func(err string, reply map[string]interface{}) {
			msf.INFO_LOG("%s response: %v %v", methodName, err, reply)
		})

		time.Sleep(time.Second)
	}
	// TestService.RpcCall(methodName, 10, float32(9.9), "abc", map[string]interface{}{"key1": 10, "key2": "def"}, []int32{123, 456}, 
	// func(err string, reply map[string]interface{}) {
	// 	msf.INFO_LOG("%s response: %v %v", methodName, err, reply)
	// })
}
