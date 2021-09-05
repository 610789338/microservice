package main


import (
    "github.com/vmihailenco/msgpack"
    "bytes"
    "fmt"
    "net"
    msf "ms_framework"
)


type C2GRouteCbInfo struct {
    rid             uint32
    connID          msf.CONN_ID
    nameSpace       string
    service         string
    rpcName         string
    createTime      int64
}

// MSG_C2G_RPC_ROUTE
type RpcC2GRpcRouteReq struct {
    NameSpace       string
    Service         string
    Rid             uint32
    IsOrdered       bool
    InnerRpc        []byte
}

type RpcC2GRpcRouteHandler struct {
    req     RpcC2GRpcRouteReq
}

func (r *RpcC2GRpcRouteHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcC2GRpcRouteHandler) GetRspPtr() interface{} {return nil}

func (r *RpcC2GRpcRouteHandler) Process(session *msf.Session) {
    /*
     * 消息路由，根据namespace:service:method从本地ip缓存中找到对应service的tcp连接，然后将消息路由过去
     * 从B里面解析出Rid
     * if Rid != 0
     *   生成GRid，并建立GRid <-> clientID:Rid的对应关系
     * 往service发送MSG_G2S_RPC_CALL请求
     */

    msf.RpcFvcCount()

    // 接口权限校验
    if msf.GetServerIdentity() == msf.SERVER_IDENTITY_CLIENT_GATE && session.GetType() == msf.SessionTcpClient {

        err := ""
        rpcPermissions, ok := gClientAccess[fmt.Sprintf("%s:%s", r.req.NameSpace, r.req.Service)]
        if !ok {
            err = fmt.Sprintf("service %s:%s not exist", r.req.NameSpace, r.req.Service)
        } else {

            var rpcName string
            decoder := msgpack.NewDecoder(bytes.NewBuffer(r.req.InnerRpc))
            decoder.Decode(&rpcName)

            access, ok := rpcPermissions[rpcName]
            if !ok || !access {
                err = fmt.Sprintf("rpc %s:%s:%s not allow", r.req.NameSpace, r.req.Service, rpcName)
            }        
        }
        
        if len(err) != 0 {   
            msf.ERROR_LOG("[rpc route] - [%s:%s] rid[%v] response[%v]", r.req.NameSpace, r.req.Service, r.req.Rid, err)

            if r.req.Rid != 0 { 
                // error response
                rpc := msf.RpcEncode(msf.MSG_G2C_RPC_RSP, r.req.Rid, err, nil)
                msg := msf.MessageEncode(rpc)
                msf.MessageSend(session.GetConn(), msg)
            }
            return
        }
    }
    
    clientID := msf.GetConnID(session.GetConn())
    remoteID := msf.GetRemoteID(r.req.NameSpace, r.req.Service)
    remote := msf.ChoiceRemote(remoteID, r.req.IsOrdered, clientID)

    if remote != nil {

        var rpcName string
        decoder := msgpack.NewDecoder(bytes.NewBuffer(r.req.InnerRpc))
        decoder.Decode(&rpcName)

        grid := uint32(0)
        if r.req.Rid != 0 {
            grid = msf.GenGid()

            cbInfo := C2GRouteCbInfo {
                rid: r.req.Rid, 
                connID: clientID, 
                nameSpace: r.req.NameSpace, 
                service: r.req.Service, 
                rpcName: rpcName, 
                createTime: msf.GetNowTimestampMs(),
            }

            // 超时时间最好大于client cb的超时时间
            msf.AddCallBack(grid, []interface{}{cbInfo}, 101, nil)
        } else {

            msf.DEBUG_LOG("[rpc route] - [%s:%s:%s] rid[%v] no response", r.req.NameSpace, r.req.Service, rpcName, r.req.Rid)
        }

        var rpc []byte
        if r.req.IsOrdered {
            rpc = msf.RpcEncode(msf.MSG_G2S_RPC_CALL_ORDERED, grid, r.req.InnerRpc)
        } else {
            rpc = msf.RpcEncode(msf.MSG_G2S_RPC_CALL, grid, r.req.InnerRpc)
        }
        msg := msf.MessageEncode(rpc)
        msf.MessageSend(remote.GetConn(), msg)

    } else {
        
        err := fmt.Sprintf("service %s:%s not exist", r.req.NameSpace, r.req.Service)
        msf.ERROR_LOG("[rpc route] - [%s:%s] rid[%v] response[%v]", r.req.NameSpace, r.req.Service, r.req.Rid, err)

        if r.req.Rid != 0 {
            // error response
            rpc := msf.RpcEncode(msf.MSG_G2C_RPC_RSP, r.req.Rid, err, nil)
            msg := msf.MessageEncode(rpc)
            msf.MessageSend(session.GetConn(), msg)
        }
    }
}

// MSG_S2G_RPC_RSP
type RpcS2GRpcRsp struct {
    GRid         uint32
    Error        string
    Reply        map[string]interface{}
}

type RpcS2GRpcRspHandler struct {
    req     RpcS2GRpcRsp
}

func (r *RpcS2GRpcRspHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcS2GRpcRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcS2GRpcRspHandler) Process(session *msf.Session) {
    // 根据GRid找到clientID:Rid，将GRid替换成Rid，然后把Error和Reply透传给client

    cbs := msf.GetCallBack(r.req.GRid)
    if nil == cbs {
        return
    }

    cbInfo := cbs[0].(C2GRouteCbInfo)

    rid, connID := cbInfo.rid, cbInfo.connID

    var conn net.Conn
    if client := msf.GetTcpClient(connID); client != nil {
        conn = client.GetConn()
    } else if remote := msf.GetRemote(connID); remote != nil {
        conn = remote.GetConn()
    }

    if nil == conn {
        msf.ERROR_LOG("[rpc route] - response error: connID[%s] not exist", connID)
        return
    }

    msf.DEBUG_LOG("[rpc route] - [%s:%s:%s] rid[%v] err[%s] reply[%v] timeCost[%vms]", 
        cbInfo.nameSpace, cbInfo.service, cbInfo.rpcName, rid, r.req.Error, r.req.Reply, msf.GetNowTimestampMs() - cbInfo.createTime)

    rpc := msf.RpcEncode(msf.MSG_G2C_RPC_RSP, rid, r.req.Error, r.req.Reply)
    msg := msf.MessageEncode(rpc)
    msf.MessageSend(conn, msg)
}

// MSG_GATE_LOGIN
type RpcGateLoginReq struct {
    ClientID    string
}

type RpcGateLoginRsp struct {
    Success     bool
}

type RpcGateLoginHandler struct {
    req     RpcGateLoginReq
    rsp     RpcGateLoginRsp
}

func (r *RpcGateLoginHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcGateLoginHandler) GetRspPtr() interface{} {return &(r.rsp)}

func (r *RpcGateLoginHandler) Process(session *msf.Session) {
    r.rsp.Success = true

    key := ""
    if "ServiceGate" == msf.GlobalCfg.Service {
        key = fmt.Sprintf("s_%s", r.req.ClientID)
    } else if "ClientGate" == msf.GlobalCfg.Service {
        key = fmt.Sprintf("c_%s", r.req.ClientID)
    } else {
        msf.ERROR_LOG("error service name %s", msf.GlobalCfg.Service)
        return
    }

    msf.INFO_LOG("[login] %s:%s", key, msf.GetConnID(session.GetConn()))

    redisCluster := msf.GetRedisCluster("myRedis2")
    value := fmt.Sprintf("%s/%s", msf.GetTcpServer().GetListerAddr().String(), msf.GetConnID(session.GetConn()))
    _, err := redisCluster.Set(key, value, 0).Result()
    if err != nil {
        msf.ERROR_LOG("[login] redis error %v - %s:%s", err, key, msf.GetConnID(session.GetConn()))
        r.rsp.Success = false
        return
    }
}

// MSG_GATE_LOGOFF
type RpcGateLogoffReq struct {
    ClientID    string
}

type RpcGateLogoffRsp struct {
    Success     bool
}

type RpcGateLogoffHandler struct {
    req     RpcGateLogoffReq
    rsp     RpcGateLogoffRsp
}

func (r *RpcGateLogoffHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcGateLogoffHandler) GetRspPtr() interface{} {return &(r.rsp)}

func (r *RpcGateLogoffHandler) Process(session *msf.Session) {
    r.rsp.Success = true

    key := ""
    if "ServiceGate" == msf.GlobalCfg.Service {
        key = fmt.Sprintf("s_%s", r.req.ClientID)
    } else if "ClientGate" == msf.GlobalCfg.Service {
        key = fmt.Sprintf("c_%s", r.req.ClientID)
    } else {
        msf.ERROR_LOG("error service name %s", msf.GlobalCfg.Service)
        return
    }

    msf.INFO_LOG("[logoff] %s:%s", key, msf.GetConnID(session.GetConn()))

    redisCluster := msf.GetRedisCluster("myRedis2")
    _, err := redisCluster.Del(key).Result()
    if err != nil {
        msf.ERROR_LOG("[logoff] redis error %v - %s:%s", err, key)
        r.rsp.Success = false
        return
    }
}


// MSG_P2G_REQ_LISTENADDR
type RpcReqListenAddrRsp struct {
    Addr     string
}

type RpcReqListenAddrHandler struct {
    rsp     RpcReqListenAddrRsp
}

func (r *RpcReqListenAddrHandler) GetReqPtr() interface{} {return nil}
func (r *RpcReqListenAddrHandler) GetRspPtr() interface{} {return &(r.rsp)}

func (r *RpcReqListenAddrHandler) Process(session *msf.Session) {

    rpc := msf.RpcEncode(msf.MSG_G2P_RSP_LISTENADDR, msf.GetTcpServer().GetListerAddr().String())
    msg := msf.MessageEncode(rpc)
    msf.MessageSend(session.GetConn(), msg)
}


// MSG_P2G_PUSH
type RpcP2GPushReq struct {
    ConnID        string
    Pid           []byte
    Namespace     string
    InnerRpc      []byte
}

type RpcP2GPushHandler struct {
    req     RpcP2GPushReq
}

func (r *RpcP2GPushHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcP2GPushHandler) GetRspPtr() interface{} {return nil}

func (r *RpcP2GPushHandler) Process(session *msf.Session) {
    
    client := msf.GetTcpClient(msf.CONN_ID(r.req.ConnID))
    if nil == client {
        msf.ERROR_LOG("[push2client] get client(%s) not exist", r.req.ConnID)
        return
    }

    msf.DEBUG_LOG("[push2client] connID - %s client - %s", r.req.ConnID, client.RemoteAddr())

    rpc := msf.RpcEncode(msf.MSG_G2C_PUSH, r.req.Pid, r.req.Namespace, r.req.InnerRpc)
    msg := msf.MessageEncode(rpc)
    msf.MessageSend(client.GetConn(), msg)
}


// MSG_C2G_VERTIFY
type RpcC2GVertifyReq struct {
    SecretKey   string
}

type RpcC2GVertifyHandler struct {
    req         RpcC2GVertifyReq
}

func (r *RpcC2GVertifyHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcC2GVertifyHandler) GetRspPtr() interface{} {return nil}

func (r *RpcC2GVertifyHandler) Process(session *msf.Session) {
    if r.req.SecretKey != msf.VERTIFY_SECRET_KEY {
        msf.ERROR_LOG("[c2g vertify] tcp client %s vertify key error %s != %s", msf.GetConnID(session.GetConn()), r.req.SecretKey, msf.VERTIFY_SECRET_KEY)
        return
    }

    connID := msf.GetConnID(session.GetConn())
    client := msf.GetTcpClient(connID)
    if nil == client {
        msf.ERROR_LOG("[c2g vertify] tcp client %s not exist", connID)
        return
    }

    msf.INFO_LOG("[c2g vertify] tcp client %s vertify %s success", connID, r.req.SecretKey)
    client.SetState(msf.TcpClientState_OK)
}


// MSG_S2G_RPC_ACCESS_REPORT
type RpcS2GRpcAccessReportReq struct {
    ServiceKey      string
    Access          map[string]bool
}

type RpcS2GRpcAccessReportHandler struct {
    req         RpcS2GRpcAccessReportReq
}

func (r *RpcS2GRpcAccessReportHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcS2GRpcAccessReportHandler) GetRspPtr() interface{} {return nil}

func (r *RpcS2GRpcAccessReportHandler) Process(session *msf.Session) {
    gClientAccess[r.req.ServiceKey] = r.req.Access
    msf.DEBUG_LOG("[s2g rpc access report] %s: %+v", r.req.ServiceKey, r.req.Access)
}
