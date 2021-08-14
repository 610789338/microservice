package main


import (
    msf "ms_framework"
    "fmt"
    "strings"
)


// MSG_G2P_RSP_LISTENADDR
type RpcRspListenAddrReq struct {
    Addr    string
}

type RpcRspListenAddrHandler struct {
    req     RpcRspListenAddrReq
}

func (r *RpcRspListenAddrHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcRspListenAddrHandler) GetRspPtr() interface{} {return nil}

func (r *RpcRspListenAddrHandler) Process(session *msf.Session) {
    gGateAddrMap[r.req.Addr] = msf.GetConnID(session.GetConn())

    msf.DEBUG_LOG("gate find addr %s", r.req.Addr)
}


// MSG_PUSH_UNSAFE
type RpcPushUnsafeReq struct {
    ClientID    string
    Typ         string
    Msg         []byte
}

type RpcPushUnsafeHandler struct {
    req     RpcPushUnsafeReq
}

func (r *RpcPushUnsafeHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcPushUnsafeHandler) GetRspPtr() interface{} {return nil}

func (r *RpcPushUnsafeHandler) Process(session *msf.Session) {

    key := ""
    if "server" == r.req.Typ {
        key = fmt.Sprintf("s_%s", r.req.ClientID)
    } else if "client" == r.req.Typ {
        key = fmt.Sprintf("c_%s", r.req.ClientID)
    } else {
        msf.ERROR_LOG("error req type %s", r.req.Typ)
        return
    }

    redisCluster := msf.GetRedisCluster("myRedis2")
    target, err := redisCluster.Get(key).Result()
    if err != nil {
        msf.ERROR_LOG("redis get %s error %v", key, err)
        return
    }

    v := strings.Split(target, "/")
    gateAddr := v[0]
    clientConnID := v[1]

    connID, ok := gGateAddrMap[gateAddr]
    if !ok {
        msf.ERROR_LOG("gate port(%s) not exist", gateAddr)
        return
    }
    
    rpc := msf.RpcEncode(msf.MSG_P2G_PUSH, clientConnID, r.req.Msg)
    msg := msf.MessageEncode(rpc)
    client := msf.GetTcpClient(msf.CONN_ID(connID))
    if nil == client {
        msf.ERROR_LOG("gate client(%s) not exist", connID)
        return
    }

    msf.MessageSend(client.GetConn(), msg)
}


// MSG_PUSH_SAFE
// TODO
