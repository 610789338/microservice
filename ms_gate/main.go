// author: youjun
// date: 2021-03-11

package main


import (
	msf "ms_framework"
	"time"
)

var gCbMap map[uint32] []interface{}
var gCbMapMaxSize = 100
var gCbChan chan []interface{}

// 利用time.After实现callback的超时控制，避免gCbMap被撑爆
func CallBackTimeOut(grid uint32) {
	select {
	case <- time.After(time.Second * 20):
		gCbChan <- []interface{}{"get&del", grid, nil}
	}
}

func CallBackMgr() {

	for true {
		elem := <- gCbChan
		oper := elem[0].(string)

		if "add" == oper {
			grid := elem[1].(uint32)
			rid := elem[2].(uint32)
			clientID := elem[3].(msf.CLIENT_ID)

			if len(gCbMap) >= gCbMapMaxSize {
				msf.ERROR_LOG("call back cache size %v > %v", len(gCbMap), gCbMapMaxSize)
				return
			}

			gCbMap[grid] = []interface{}{rid, clientID}

			go CallBackTimeOut(grid)

		} else if "get&del" == oper {
			grid := elem[1].(uint32)
			rcID, ok := gCbMap[grid]

			rcIDChan := elem[2]
			if ok {
				if rcIDChan != nil {
					rcIDChan.(chan []interface{}) <- rcID
				}
				delete(gCbMap, grid)

			} else {
				if rcIDChan != nil {
					rcIDChan.(chan []interface{}) <- nil
					msf.ERROR_LOG("call back get error %v", grid)
				}
			}
		}
	}
}

func CallBackMgrStart() {
	gCbMap = make(map[uint32] []interface{})
	gCbChan = make(chan []interface{})
	go CallBackMgr()
}


func main() {
	CallBackMgrStart()

	msf.Init("127.0.0.1", 8888)
	msf.RegistRpcHandler(msf.MSG_C2G_RPC_ROUTE, 	func() msf.RpcHandler {return new(RpcC2GRpcRouteHandler)})
	msf.RegistRpcHandler(msf.MSG_COMMON_RSP, 		func() msf.RpcHandler {return new(RpcS2GCommonRspHandler)})

	msf.OnRemoteDiscover("YJ", "testService", "127.0.0.1", 6666)
	msf.Start()
}
