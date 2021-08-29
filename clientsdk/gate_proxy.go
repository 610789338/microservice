package clientsdk

import (
    "net"
    "fmt"
    msf "ms_framework"
    "io"
    "reflect"
    // "sync"
    "github.com/vmihailenco/msgpack"
    "bytes"
)


type CallBack func(err string, result map[string]interface{})

type GateProxy struct {
    ip              string
    port            int
    conn            net.Conn
    recvBuf         []byte
    remainLen       uint32
}

func (g *GateProxy) Start() {
    go g.HandleRead()
}

func (g *GateProxy) HandleRead() {
    defer func() {
        msf.INFO_LOG("tcp client close %v", g.conn.RemoteAddr())
        g.conn.Close()
    } ()

    for true {
        len, err := g.conn.Read(g.recvBuf[g.remainLen:])
        if err != nil {
            if err != io.EOF {
                msf.ERROR_LOG("read error %v", err)
                break
            }
        }

        if 0 == len {
            // remote close
            msf.INFO_LOG("tcp connection close by remote %v %v", g.conn.RemoteAddr(), err)
            break
        }

        g.remainLen += uint32(len)
        if g.remainLen > msf.RECV_BUF_MAX_LEN/2 {
            msf.WARN_LOG("tcp connection buff cache too long!!! %dk > %dk", g.remainLen/1024, msf.RECV_BUF_MAX_LEN/1024)
            
        } else if g.remainLen > msf.RECV_BUF_MAX_LEN {
            msf.ERROR_LOG("tcp connection buff cache overflow!!! %dk > %dk", g.remainLen/1024, msf.RECV_BUF_MAX_LEN/1024)
            break
        }

        procLen := msf.GetRpcMgr().MessageDecode(g.Turn2Session(), g.recvBuf[:g.remainLen])
        g.remainLen -= procLen
        if g.remainLen < 0 {
            msf.ERROR_LOG("g.remainLen(%d) < 0 procLen(%d) @%s", g.remainLen, procLen, g.conn.RemoteAddr())
            g.remainLen = 0
            continue
        }

        copy(g.recvBuf, g.recvBuf[procLen: procLen + g.remainLen])
    }
}

func (g *GateProxy) Login(clientID string, namespace string) {
    g.RpcCall(msf.MSG_GATE_LOGIN, clientID)

    innerRpc := msf.GetRpcMgr().RpcEncode(msf.MSG_PUSH_RESTORE, clientID)
    rpc := msf.GetRpcMgr().RpcEncode(msf.MSG_C2G_RPC_ROUTE, namespace, "PushService", 0, false, innerRpc)
    msg := msf.GetRpcMgr().MessageEncode(rpc)
    msf.MessageSend(g.conn, msg)
}

func (g *GateProxy) Logoff(clientID string) {
    g.RpcCall(msf.MSG_GATE_LOGOFF, clientID)
}

func (g *GateProxy) RpcCall(rpcName string, args ...interface{}) {
    rpc := msf.GetRpcMgr().RpcEncode(rpcName, args...)
    msg := msf.GetRpcMgr().MessageEncode(rpc)
    msf.MessageSend(g.conn, msg)
}

func (g *GateProxy) Turn2Session() *msf.Session {
    return msf.CreateSession(msf.SessionRemote, g.conn)
}

func (g *GateProxy) LocalAddr() string {
    return g.conn.LocalAddr().String()
}

func CreateGateProxy(_ip string, _port int) *GateProxy {
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", _ip, _port))
    if err != nil {
        msf.ERROR_LOG("connect %s:%d error %v", _ip, _port, err)
        return nil
    }

    msf.INFO_LOG("connect %s:%d success %v", _ip, _port, conn)

    g := &GateProxy {
        ip:   _ip,
        port: _port,
        conn: conn, 
        recvBuf: make([]byte, msf.RECV_BUF_MAX_LEN), 
        remainLen: 0,
    }

    g.RpcCall(msf.MSG_C2G_VERTIFY, msf.VERTIFY_SECRET_KEY)
    g.Start()

    return g
}

func (g *GateProxy) CreateServiceProxy(namespace string, serviceName string) *ServiceProxy {
    return &ServiceProxy{Gp: g, Namespace: namespace, ServiceName: serviceName}
}


type ServiceProxy struct {
    Gp                *GateProxy
    Namespace         string
    ServiceName       string
}

// c2s的rpc调用，最后一个参数若是CallBack，则建立rid<->callback的缓存
func (sp *ServiceProxy) RpcCall(rpcName string, args ...interface{}) {

    var rid uint32 = 0
    if len(args) > 0 {
        lastArg := args[len(args)-1]
        // t := reflect.TypeOf(lastArg)
        cb, ok := lastArg.(CallBack)
        if ok {
            args = args[:len(args)-1]
            if cb != nil {
                // msf.DEBUG_LOG("rpc last arg is %v %T %v", lastArg, lastArg, cb)
                rid = msf.GenGid()

                timeoutCb := func() {
                    error := fmt.Sprintf("rpc call %s:%s:%s time out", sp.Namespace, sp.ServiceName, rpcName)
                    lastArg.(CallBack)(error, nil)
                }

                msf.AddCallBack(rid, []interface{}{lastArg.(CallBack)}, 100, timeoutCb)
            }
        }
    }

    // msf.DEBUG_LOG("rpc call %s args %v rid %d", rpcName, args, rid)

    // TODO: 这个接口再封装一下
    innerRpc := msf.GetRpcMgr().RpcEncode(rpcName, args...)
    sp.Gp.RpcCall(msf.MSG_C2G_RPC_ROUTE, sp.Namespace, sp.ServiceName, rid, true, innerRpc)
}

// MSG_G2C_RPC_RSP
type RpcG2CRpcRspReq struct {
    Rid     uint32
    Error   string
    Reply   map[string]interface{}
}

type RpcG2CRpcRspHandler struct {
    req     RpcG2CRpcRspReq
}

func (r *RpcG2CRpcRspHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcG2CRpcRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcG2CRpcRspHandler) Process(session *msf.Session) {

    cbs := msf.GetCallBack(r.req.Rid)
    if nil == cbs {
        msf.ERROR_LOG("RpcG2CRpcRspHandler GetCallBack error %v maybe timeout", r.req.Rid)
        return
    }

    cb := cbs[0].(CallBack)
    if cb != nil {
        cb(r.req.Error, r.req.Reply)
    }
}

// MSG_G2C_PUSH
type RpcG2CPushReq struct {
    Pid             []byte
    Namespace       string
    InnerRpc        []byte
}

type RpcG2CPushHandler struct {
    req     RpcG2CPushReq
}

func (r *RpcG2CPushHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcG2CPushHandler) GetRspPtr() interface{} {return nil}

func (r *RpcG2CPushHandler) Process(session *msf.Session) {

    // msf.DEBUG_LOG("---for debug %d", len(r.req.InnerRpc))
    decoder := msgpack.NewDecoder(bytes.NewBuffer(r.req.InnerRpc))

    var rpcName string
    decoder.Decode(&rpcName)

    handlerGen, ok := msf.GetRpcMgr().GetRpcHanderGenerator(rpcName)
    if !ok {
        msf.ERROR_LOG("rpc %s not exist", rpcName)
        return
    }

    handler := handlerGen()
    reqPtr := reflect.ValueOf(handler.GetReqPtr())
    if handler.GetReqPtr() != nil && !reqPtr.IsNil() {
        stValue := reqPtr.Elem()
        for i := 0; i < stValue.NumField(); i++ {
            nv := reflect.New(stValue.Field(i).Type())
            if err := decoder.Decode(nv.Interface()); err != nil {
                msf.ERROR_LOG("rpc %s arg %s(%v) decode error: %v", rpcName, stValue.Type().Field(i).Name, nv.Type(), err)
                return
            }

            stValue.Field(i).Set(nv.Elem())
        }
    }

    handler.Process(session)

    if len(r.req.Pid) != 0 {
        innerRpc := msf.GetRpcMgr().RpcEncode(msf.MSG_PUSH_REPLY, r.req.Pid)
        rpc := msf.GetRpcMgr().RpcEncode(msf.MSG_C2G_RPC_ROUTE, r.req.Namespace, "PushService", 0, false, innerRpc)
        msg := msf.GetRpcMgr().MessageEncode(rpc)
        msf.MessageSend(session.GetConn(), msg)
    }
}

func init() {
    msf.CreateSimpleRpcMgr()
    msf.RegistRpcHandlerForce(msf.MSG_G2C_RPC_RSP,      func() msf.RpcHandler {return new(RpcG2CRpcRspHandler)})
    msf.RegistRpcHandlerForce(msf.MSG_G2C_PUSH,         func() msf.RpcHandler {return new(RpcG2CPushHandler)})
    msf.StartTaskPool()
}
