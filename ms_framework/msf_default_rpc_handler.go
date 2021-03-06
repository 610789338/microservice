package ms_framework

import (
    "fmt"
    "github.com/vmihailenco/msgpack"
    "bytes"
    "reflect"
    "runtime"
)


var MSG_C2G_RPC_ROUTE           = "a"  // client to gate rpc route (client include mservice/gameserver/gameclient)
var MSG_G2S_RPC_CALL            = "b"  // gate to service rpc call
var MSG_S2G_RPC_RSP             = "c"  // service to gate rpc response
var MSG_G2C_RPC_RSP             = "d"  // gate to client rpc response


var MSG_G2S_RPC_CALL_ORDERED    = "t"  // gate to service rpc call ordered

var MSG_HEART_BEAT_REQ          = "e"  // heart beat request
var MSG_HEART_BEAT_RSP          = "f"  // heart beat response

var MSG_C2G_VERTIFY             = "g"  // client to gate vertify

var MSG_GATE_LOGIN              = "h"  // client login from gate
var MSG_GATE_LOGOFF             = "i"  // client logoff from gate

var MSG_P2G_REQ_LISTENADDR      = "j"  // push service to gate request listen addr
var MSG_G2P_RSP_LISTENADDR      = "k"  // gate service to push response listen addr

var MSG_S2P_PUSH                = "l"  // service to push service
var MSG_P2G_PUSH                = "m"  // push service to gate
var MSG_G2C_PUSH                = "n"  // gate push to client
var MSG_PUSH_REPLY              = "o"  // push reply (for safe push)
var MSG_PUSH_RESTORE            = "p"  // push record restore (when client login)

var MSG_S2G_RPC_ACCESS_REPORT   = "r"  // service to gate rpc access report



// MSG_G2S_RPC_CALL
type RpcG2SRpcCallReq struct {
    GRid            uint32
    InnerRpc        []byte  // 
}

type RpcG2SRpcCallHandler struct {
    req             RpcG2SRpcCallReq
}

func (r *RpcG2SRpcCallHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcG2SRpcCallHandler) GetRspPtr() interface{} {return nil}

func (r *RpcG2SRpcCallHandler) Process(session *Session) {
    
    RpcFvcCount()
    
    error, reply := r.rpcHandle(session)

    if r.req.GRid != 0 {

        // response
        rpc := rpcMgr.RpcEncode(MSG_S2G_RPC_RSP, r.req.GRid, error, reply)
        msg := rpcMgr.MessageEncode(rpc)
        MessageSend(session.conn, msg)
    }

    if error != "" {
        ERROR_LOG("[rpc call] - %s", error)
    }
}

func (r *RpcG2SRpcCallHandler) rpcHandle(session *Session) (string, map[string]interface{}){

    decoder := msgpack.NewDecoder(bytes.NewBuffer(r.req.InnerRpc))

    var rpcName string
    decoder.Decode(&rpcName)

    handlerGen, ok := rpcMgr.GetRpcHanderGenerator(rpcName)
    if !ok {
        return fmt.Sprintf("rpc %s not exist", rpcName), nil
    }

    handler := handlerGen()
    reqPtr := reflect.ValueOf(handler.GetReqPtr())
    if handler.GetReqPtr() != nil && !reqPtr.IsNil() {
        stValue := reqPtr.Elem()
        for i := 0; i < stValue.NumField(); i++ {
            nv := reflect.New(stValue.Field(i).Type())
            if err := decoder.Decode(nv.Interface()); err != nil {
                return fmt.Sprintf("rpc %s arg %s(%v) decode error: %v", rpcName, stValue.Type().Field(i).Name, nv.Type(), err), nil
            }

            stValue.Field(i).Set(nv.Elem())
        }
    }

    r.doProcessWithPanicProtect(handler, session)

    if r.req.GRid != 0 {
        rspPtr := reflect.ValueOf(handler.GetRspPtr())
        if handler.GetRspPtr() == nil || rspPtr.IsNil() {
            return fmt.Sprintf("rpc %s need response but get nil", rpcName), nil
        }

        // for response
        stMap := make(map[string]interface{})
        stValue := rspPtr.Elem()
        for i := 0; i < stValue.NumField(); i++ {
            stMap[stValue.Type().Field(i).Name] = stValue.Field(i).Interface()
        }

        err := GetResponseErr(session)
        DEBUG_LOG("[rpc call] - [%s] args[%v] err[%v] reply[%v]", rpcName, handler.GetReqPtr(), err, stMap)
        return err, stMap
    }

    DEBUG_LOG("[rpc call] - [%s] args[%v] reply[nil]", rpcName, handler.GetReqPtr())
    return "", nil
}

func (r *RpcG2SRpcCallHandler) doProcessWithPanicProtect(handler RpcHandler, session *Session) {
    defer func() {
        panicMsg := ""
        if err := recover(); err != nil {
            panicMsg = fmt.Sprint(err);
        }

        if len(panicMsg) != 0 {
            buf := make([]byte, 2048)
            n := runtime.Stack(buf, false)
            stackInfo := fmt.Sprintf("%s", buf[:n])
            ERROR_LOG(panicMsg)
            ERROR_LOG("panic stack info %s", stackInfo)
        }
    }()

    handler.Process(session)
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

func (r *RpcG2CRpcRspHandler) Process(session *Session) {

    cbs := GetCallBack(r.req.Rid)
    if nil == cbs {
        ERROR_LOG("RpcG2CRpcRspHandler GetCallBack error %v", r.req.Rid)
        return
    }

    ch := cbs[0].(chan []interface{})
    ch <- []interface{}{r.req.Error, r.req.Reply}
}

// ***************************  heart beat ***************************

// MSG_HEART_BEAT_REQ
type RpcHeartBeatReqHandler struct {
}

func (r *RpcHeartBeatReqHandler) GetReqPtr() interface{} {return nil}
func (r *RpcHeartBeatReqHandler) GetRspPtr() interface{} {return nil}

func (r *RpcHeartBeatReqHandler) Process(session *Session) {
    // DEBUG_LOG("heart beat request from %v", GetConnID(session.conn))

    rpc := RpcEncode(MSG_HEART_BEAT_RSP)
    msg := MessageEncode(rpc)
    MessageSend(session.conn, msg)
}

// MSG_HEART_BEAT_RSP
type RpcHeartBeatRspHandler struct {
}

func (r *RpcHeartBeatRspHandler) GetReqPtr() interface{} {return nil}
func (r *RpcHeartBeatRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcHeartBeatRspHandler) Process(session *Session) {
    // DEBUG_LOG("heart beat response from %v", GetConnID(session.conn))
}
