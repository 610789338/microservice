package ms_framework

import (
	"time"
	"sync"
)

var gCbMap = make(map[uint32] []interface{})
var cbMutex sync.Mutex

// 利用time.After实现callback的超时控制，避免gCbMap被撑爆
func CallBackTimeOut(cbid uint32, waitTime int8, timeOutCb func()) {
	select {
	case <- time.After(time.Second * time.Duration(waitTime)):
		cbMutex.Lock()
		_, ok := gCbMap[cbid]
		if ok {
			ERROR_LOG("call back timeout %v", cbid)
			if timeOutCb != nil {
				timeOutCb()
			}
		}

		delete(gCbMap, cbid)
		cbMutex.Unlock()
	}
}

func AddCallBack(cbid uint32, cbs []interface{}, waitTime int8, timeOutCb func()) {
	cbMutex.Lock()
	gCbMap[cbid] = cbs

	// _, ok := gCbMap[cbid]
	// ERROR_LOG("add call back %v %v, %v", cbid, rid, ok)

	cbMutex.Unlock()

	// TODO：协程开多了会影响性能，可以考虑用排序链表来实现
	go CallBackTimeOut(cbid, waitTime, timeOutCb)
}

func GetCallBack(cbid uint32) []interface{} {
	cbMutex.Lock()
	cbs, ok := gCbMap[cbid]
	delete(gCbMap, cbid)
	cbMutex.Unlock()

	if ok {
		return cbs
	} else {
		ERROR_LOG("call back get error %v", cbid)
		return nil
	}
}
