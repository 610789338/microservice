package main

import (
	msf "ms_framework"
	"time"
	"sync"
)

var gCbMap = make(map[uint32] []interface{})
var gCbMapMaxSize = 10000
var cbMutex sync.Mutex


// 利用time.After实现callback的超时控制，避免gCbMap被撑爆
func CallBackTimeOut(grid uint32) {
	select {
	case <- time.After(time.Second * 20):
		cbMutex.Lock()
		_, ok := gCbMap[grid]
		if ok {
			msf.ERROR_LOG("call back timeout %v", grid)
		}

		delete(gCbMap, grid)
		cbMutex.Unlock()
	}
}

func AddCallBack(grid uint32, rid uint32, clientID msf.CLIENT_ID) {
	cbMutex.Lock()
	gCbMap[grid] = []interface{}{rid, clientID}

	// _, ok := gCbMap[grid]
	// msf.ERROR_LOG("add call back %v %v, %v", grid, rid, ok)

	cbMutex.Unlock()

	// 协程开多了会影响性能，可以考虑用排序链表来实现
	go CallBackTimeOut(grid)

}

func GetCallBack(grid uint32) (uint32, msf.CLIENT_ID){
	cbMutex.Lock()
	rcID, ok := gCbMap[grid]
	delete(gCbMap, grid)
	cbMutex.Unlock()

	if ok {
		return rcID[0].(uint32), rcID[1].(msf.CLIENT_ID)
	} else {
		msf.ERROR_LOG("call back get error %v", grid)
		return 0, ""
	}
}
