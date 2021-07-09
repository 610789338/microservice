package ms_framework

import (
    "math/rand"
    "sort"
    "sync"
)


const (
    BalanceStrategy_Rand         int8 = iota       // 随机
    BalanceStrategy_RoundRobin                     // 轮询调度
    BalanceStrategy_TODO                           // 基于启动时序的负载均衡 - 新进成员承担更多负载
)

var BALANCE_STRATEGY int8 = BalanceStrategy_RoundRobin

type BalanceInfo struct {
    weight             uint32
}

type LoadBalancer struct {
    elements         map[string]*BalanceInfo       // target:weight
    rrbIdx           uint16                        // for round robin

    mutex            sync.RWMutex
}

func (l *LoadBalancer) AddElement(ele string) bool {
    l.mutex.Lock()
    defer l.mutex.Unlock()

    if nil == l.elements {
        l.elements = make(map[string]*BalanceInfo)
    }

    _, ok := l.elements[ele]
    if ok {
        ERROR_LOG("[LoadBalancer] add element error %s already exist", ele)
        return false
    }

    l.elements[ele] = &BalanceInfo{weight: 0}

    return true
}

func (l *LoadBalancer) DelElement(ele string) bool {
    l.mutex.Lock()
    defer l.mutex.Unlock()

    _, ok := l.elements[ele]
    if !ok {
        ERROR_LOG("[LoadBalancer] del element error %s not exist", ele)
        return false
    }

    delete(l.elements, ele)

    return true
}

func (l *LoadBalancer) LoadBalance() (ele string) {

    switch BALANCE_STRATEGY {
    case BalanceStrategy_Rand:
        ele = l.LoadBalanceRand()

    case BalanceStrategy_RoundRobin:
        ele = l.LoadBalanceRoundRobin()

    default:
        ele = l.LoadBalanceRoundRobin()
    }

    l.mutex.Lock()
    defer l.mutex.Unlock()

    if element, ok := l.elements[ele]; ok {
        element.weight += 1
    }

    return
}

func (l *LoadBalancer) LoadBalanceRand() (ele string) {
    l.mutex.RLock()
    defer l.mutex.RUnlock()

    if len(l.elements) == 0 {
        return
    }

    m := make([]string, 0, len(l.elements))

    for ele, _ = range l.elements {
        m = append(m, ele)
    }

    sort.Strings(m)

    idx := rand.Intn(len(m))
    ele = m[idx]

    return
}

func (l *LoadBalancer) LoadBalanceRoundRobin() (ele string) {
    l.mutex.RLock()

    if len(l.elements) == 0 {
        l.mutex.RUnlock()
        return
    }

    m := make([]string, 0, len(l.elements))

    for ele, _ = range l.elements {
        m = append(m, ele)
    }

    sort.Strings(m)

    l.mutex.RUnlock()


    l.mutex.Lock()
    defer l.mutex.Unlock()

    if l.rrbIdx >= uint16(len(m)) {
        l.rrbIdx = 0
    }

    ele = m[l.rrbIdx]
    l.rrbIdx += 1

    return
}

// func init() {

//     lbRandTest()
//     lbRoundRobinTest()
// }

func lbRandTest() {
    BALANCE_STRATEGY = BalanceStrategy_Rand
    lb := LoadBalancer{}
    lb.AddElement("e1")
    lb.AddElement("e2")
    lb.AddElement("e3")
    lb.AddElement("e4")
    lb.AddElement("e5")

    loop := 1000
    for loop > 0 {
        loop -= 1
        lb.LoadBalance()
    }

    INFO_LOG("lbRandTest info 1: %v %+v", len(lb.elements), lb)

    lb.DelElement("e4")
    lb.DelElement("e5")
    lb.AddElement("e6")

    loop = 1000
    for loop > 0 {
        loop -= 1
        lb.LoadBalance()
    }

    INFO_LOG("lbRandTest info 2: %v %+v", len(lb.elements), lb)
}

func lbRoundRobinTest() {
    BALANCE_STRATEGY = BalanceStrategy_RoundRobin
    lb := LoadBalancer{}
    lb.AddElement("e1")
    lb.AddElement("e2")
    lb.AddElement("e3")
    lb.AddElement("e4")
    lb.AddElement("e5")

    loop := 1000
    for loop > 0 {
        loop -= 1
        lb.LoadBalance()
    }

    INFO_LOG("lbRoundRobinTest info 1: %v %+v", len(lb.elements), lb)

    lb.DelElement("e4")
    lb.DelElement("e5")
    lb.AddElement("e6")

    loop = 1000
    for loop > 0 {
        loop -= 1
        lb.LoadBalance()
    }

    INFO_LOG("lbRoundRobinTest info 2: %v %+v", len(lb.elements), lb)
}
