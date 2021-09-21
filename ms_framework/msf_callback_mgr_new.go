package ms_framework

import (
    "time"
    "sync"
)

var gCbMap = make(map[uint32] []interface{})
var cbMutex sync.Mutex

// 利用优先队列实现callback的超时控制，避免gCbMap被撑爆
type Node struct {
    cbid        uint32
    endTime     int64
    timeOutCb   func()
    prev        *Node
    next        *Node
}

type PriorityQueue struct {
    firstNode   *Node
    lastNode    *Node
    size        int32
}

var gPriorityQueue PriorityQueue
var chQueueAdd chan *Node = make(chan *Node)
var chQueueDel chan *Node = make(chan *Node)

func StartCallbackMgr() {

    ch := time.After(time.Second * 1)

    for {
        select {
            case node := <- chQueueAdd:
                queueAdd(node)
            case node := <- chQueueDel:
                queueDel(node)
            case _ = <- ch:
                priorityQueueTick()
                ch = time.After(time.Second * 1)
        }
    }
}

func queueAdd(node *Node) *Node {
    gPriorityQueue.size += 1

    if gPriorityQueue.firstNode == nil {
        gPriorityQueue.firstNode = node
        gPriorityQueue.lastNode = node
        return node
    }

    cur := gPriorityQueue.firstNode
    for true {
        next := cur.next

        if next == nil {
            cur.next = node
            node.prev = cur
            gPriorityQueue.lastNode = node
            break
        }

        // 后进来的node.endTime一般来说会更大，逆序排序可以减少for次数
        if node.endTime >= next.endTime {
            cur.next = node
            node.prev = cur
            node.next = next
            next.prev = node
            break
        }

        cur = next
    }

    return node
}

func queueDel(node *Node) {
    prev := node.prev
    next := node.next

    if prev != nil {
        prev.next = next
    }

    if next != nil {
        next.prev = prev
    }

    node.prev = nil
    node.next = nil

    if node == gPriorityQueue.firstNode {
        gPriorityQueue.firstNode = next
    }

    if node == gPriorityQueue.lastNode {
        gPriorityQueue.lastNode = prev
    }

    gPriorityQueue.size -= 1
}

func priorityQueueTick() {
    now := GetNowTimestamp()
    cur := gPriorityQueue.lastNode
    for true {
        if cur == nil {
            break
        }

        prev := cur.prev

        if cur.endTime <= now {
            
            cbMutex.Lock()
            _, ok := gCbMap[cur.cbid]
            delete(gCbMap, cur.cbid)
            cbMutex.Unlock()

            if ok {
                ERROR_LOG("call back timeout %v", cur.cbid)
                if cur.timeOutCb != nil {
                    cur.timeOutCb()
                }
            }

            queueDel(cur)
        } else {
            break
        }

        cur = prev
    }
    
    if gPriorityQueue.size != 0 {
        INFO_LOG("[PriorityQueue] size %v", gPriorityQueue.size)   
    }
}

func AddCallBack(cbid uint32, cbs []interface{}, waitTime int8, timeOutCb func()) {
    node := Node{cbid: cbid, endTime: int64(waitTime)+GetNowTimestamp(), timeOutCb: timeOutCb, prev: nil, next: nil}
    cbMutex.Lock()
    gCbMap[cbid] = []interface{}{cbs, &node}
    cbMutex.Unlock()

    chQueueAdd <- &node
}

func GetCallBack(cbid uint32) []interface{} {
    cbMutex.Lock()
    cbsAndNode, ok := gCbMap[cbid]
    delete(gCbMap, cbid)
    cbMutex.Unlock()

    if ok {
        cbs, node := cbsAndNode[0].([]interface{}), cbsAndNode[1].(*Node)
        chQueueDel <- node
        return cbs
    } else {
        ERROR_LOG("call back get error %v", cbid)
        return nil
    }
}

func init() {
    go StartCallbackMgr()
}
