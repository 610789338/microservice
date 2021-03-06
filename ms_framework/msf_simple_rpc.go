package ms_framework

import (
    // "errors"
    "reflect"
    "github.com/vmihailenco/msgpack"
    "bytes"
    "fmt"
    "net"
)

const (
    SessionTcpClient int8 = iota
    SessionRemote
)

type Session struct {
    typ         int8
    conn        net.Conn
    err         string
}

func (session *Session) GetConn() net.Conn {
    return session.conn
}

func (session *Session) GetType() int8 {
    return session.typ
}

func CreateSession(typ int8, conn net.Conn) *Session {
    return &Session{typ: typ, conn: conn}
}

func SetResponseErr(session *Session, err string) {
    session.err = err
}

func GetResponseErr(session *Session) string {
    return session.err
}


var MAX_PACKET_SIZE uint32 = 16*1024  // 16K
var MESSAGE_SIZE_LEN uint32 = 2

func ReadPacketLen(buf []byte) uint32 {
    // return ReadUint32(buf)
    return uint32(ReadUint16(buf))
}

func WritePacketLen(buf []byte, v uint32) {
    // WriteUint32(buf, v)
    WriteUint16(buf, uint16(v))
}

type RpcHandler interface {
    GetReqPtr() interface{}
    GetRspPtr() interface{}
    Process(session *Session)
}

type RpcHanderGenerator func() RpcHandler

type RpcClientPermission interface {
    ClientAccess() bool
}

type SimpleRpcMgr struct {
    rpcs            map[string]RpcHanderGenerator
    clientAccess    map[string]bool
    stopRegist      bool
}

func (rmgr *SimpleRpcMgr) RegistRpcHandler(name string, gen RpcHanderGenerator) {
    if rmgr.stopRegist {
        panic("can not retist rpc handler now")
    }

    _, ok := rmgr.rpcs[name]
    if ok {
        panic(fmt.Sprintf("RegistRpcHandler %s repeat !!!", name))
    }

    rmgr.rpcs[name] = gen
    rmgr.GenClientAccess(name, gen)
}

func (rmgr *SimpleRpcMgr) RegistRpcHandlerForce(name string, gen RpcHanderGenerator) {
    if rmgr.stopRegist {
        panic("can not retist rpc handler now")
    }

    rmgr.rpcs[name] = gen
    rmgr.GenClientAccess(name, gen)
}

func (rmgr *SimpleRpcMgr) GenClientAccess(name string, gen RpcHanderGenerator) {
    handler := gen()
    access, ok := handler.(RpcClientPermission)
    if ok && access.ClientAccess() {
        rmgr.clientAccess[name] = true
    } else {
        rmgr.clientAccess[name] = false
    }
}

func (rmgr *SimpleRpcMgr) MessageDecode(session *Session, msg []byte) uint32 {
    var offset uint32 = 0

    var bufLen uint32 = uint32(len(msg))

    for offset < bufLen {
        if bufLen - offset < MESSAGE_SIZE_LEN {
            WARN_LOG("remain len(%d) < MESSAGE_SIZE_LEN(%d)", bufLen - offset, MESSAGE_SIZE_LEN)
            break
        }

        pkgLen := ReadPacketLen(msg[offset:])
        if bufLen - offset < MESSAGE_SIZE_LEN + pkgLen {
            WARN_LOG("remain len(%d) < MESSAGE_SIZE_LEN(%d) + pkgLen(%d)", bufLen - offset, MESSAGE_SIZE_LEN, pkgLen)
            break
        }

        offset += MESSAGE_SIZE_LEN

        if pkgLen > MAX_PACKET_SIZE {
            ERROR_LOG("packet size too long %d > %d", pkgLen, MAX_PACKET_SIZE)
        } else {
            buf := make([]byte, pkgLen)
            copy(buf, msg[offset: offset + pkgLen])
            rmgr.RpcDecode(session, buf)
        }

        offset += pkgLen
    }

    return offset
}

func (rmgr *SimpleRpcMgr) RpcDecode(session *Session, buf []byte) {

    decoder := msgpack.NewDecoder(bytes.NewBuffer(buf))

    var rpcName string
    decoder.Decode(&rpcName)

    task := func() {
        handlerGen, ok := rmgr.GetRpcHanderGenerator(rpcName)
        if !ok {
            ERROR_LOG("rpc %s not exist", rpcName)
            return
        }

        handler := handlerGen()
        if handler.GetReqPtr() != nil {
            reqPtr := reflect.ValueOf(handler.GetReqPtr())
            stValue := reqPtr.Elem()
            for i := 0; i < stValue.NumField(); i++ {
                nv := reflect.New(stValue.Field(i).Type())
                if err := decoder.Decode(nv.Interface()); err != nil {
                    ERROR_LOG("rpc(%s) arg(%s-%v) decode error: %v", rpcName, stValue.Type().Field(i).Name, nv.Type(), err)
                    return
                }

                stValue.Field(i).Set(nv.Elem())
            }
        }

        handler.Process(session)
    }

    if !rmgr.PreCheck(session, rpcName) {
        return
    }

    if rpcName == MSG_C2G_VERTIFY || rpcName == MSG_GATE_LOGIN {
        // ?????????????????????????????????????????????rpc??????????????????????????????
        task()
    } else if rpcName == MSG_G2S_RPC_CALL_ORDERED {
        // ???????????? - ????????????????????????????????????????????????
        gTaskPool.ProduceTaskSeparate(task)
    } else {
        // ??????task pool
        gTaskPool.ProduceTask(task)
    }
}

func (rmgr *SimpleRpcMgr) PreCheck(session *Session, rpcName string) bool {

    if rpcName == MSG_C2G_VERTIFY {
        return true
    }

    if session.typ != SessionTcpClient {
        return true
    }

    tcpClient := GetTcpClient(GetConnID(session.conn))
    if tcpClient.state != TcpClientState_OK {
        // ???????????????????????????
        ERROR_LOG("illegal tcp client %s, rpc - %s", GetConnID(session.conn), rpcName)
        tcpClient.SetState(TcpClientState_EXIT)
        return false
    }

    return true
}


func (rmgr *SimpleRpcMgr) IsSync(rpcName string) bool {
    if rpcName == MSG_C2G_VERTIFY || rpcName == MSG_GATE_LOGIN || rpcName == MSG_G2S_RPC_CALL_ORDERED {
        return true
    }

    return false
}

func (rmgr *SimpleRpcMgr) MessageEncode(buf []byte) []byte {

    bufLen := uint32(len(buf))
    if bufLen > MAX_PACKET_SIZE {
        panic(fmt.Sprintf("message len > %s", MAX_PACKET_SIZE))
    }
    msg := make([]byte, MESSAGE_SIZE_LEN + bufLen)
    WritePacketLen(msg, bufLen)
    copy(msg[MESSAGE_SIZE_LEN:], buf)

    return msg
}

func (rmgr *SimpleRpcMgr) RpcEncode(name string, args ...interface{}) []byte {

    writer := &bytes.Buffer{}
    encoder := msgpack.NewEncoder(writer)

    if err := encoder.Encode(name); err != nil {
        ERROR_LOG("encode rpc name error %v", err)
    }

    for _, arg := range args {
        if err := encoder.Encode(arg); err != nil {
            ERROR_LOG("args encode error %s: %v", name, err)
            continue
        }
    }

    return writer.Bytes()
}

func (rmgr *SimpleRpcMgr) GetRpcHanderGenerator(rpcName string) (RpcHanderGenerator, bool) {
    f, ok := rmgr.rpcs[rpcName]
    return f, ok
}

var rpcMgr *SimpleRpcMgr = nil

func GetRpcMgr() *SimpleRpcMgr {
    return rpcMgr
}

func CreateSimpleRpcMgr() {
    rpcMgr = &SimpleRpcMgr{rpcs: make(map[string]RpcHanderGenerator), clientAccess: make(map[string]bool), stopRegist: false}

    // default handler
    rpcMgr.RegistRpcHandler(MSG_G2S_RPC_CALL,            func() RpcHandler {return new(RpcG2SRpcCallHandler)})      // for service
    rpcMgr.RegistRpcHandler(MSG_G2S_RPC_CALL_ORDERED,    func() RpcHandler {return new(RpcG2SRpcCallHandler)})      // for service

    rpcMgr.RegistRpcHandler(MSG_G2C_RPC_RSP,             func() RpcHandler {return new(RpcG2CRpcRspHandler)})     // for client


    rpcMgr.RegistRpcHandler(MSG_HEART_BEAT_REQ,          func() RpcHandler {return new(RpcHeartBeatReqHandler)}) // for all
    rpcMgr.RegistRpcHandler(MSG_HEART_BEAT_RSP,          func() RpcHandler {return new(RpcHeartBeatRspHandler)}) // for all
}

func StopRcpHandlerRegist() {
    rpcMgr.stopRegist = true
}

func RegistRpcHandler(name string, gen RpcHanderGenerator) {
    rpcMgr.RegistRpcHandler(name, gen)
}

func RegistRpcHandlerForce(name string, gen RpcHanderGenerator) {
    rpcMgr.RegistRpcHandlerForce(name, gen)
}

func MessageEncode(b []byte) []byte {
    return rpcMgr.MessageEncode(b)
}

func RpcEncode(name string, args ...interface{}) []byte {
    return rpcMgr.RpcEncode(name, args...)
}

type RpcCallTimeOutError struct {
    err     string
}

func RpcCall(serviceName string, rpcName string, rid uint32, reSendCnt int8, args ...interface{}) (err string, reply map[string]interface{}) {

    innerRpc := rpcMgr.RpcEncode(rpcName, args...)
    rpc := rpcMgr.RpcEncode(MSG_C2G_RPC_ROUTE, GlobalCfg.Namespace, serviceName, rid, false, innerRpc)
    msg := rpcMgr.MessageEncode(rpc)

    var client *TcpClient = nil
    connID := CONN_ID(tcpServer.lb.LoadBalance())
    client, ok := tcpServer.clients[connID]
    if !ok {
        ERROR_LOG("[s2s rpc call] load balance error %s", connID)
        return
    }

    ch := make(chan []interface{})

    if rid != 0 {
        // must before MessageSend
        timeoutCb := func() {
            timeout := RpcCallTimeOutError{err: fmt.Sprintf("s2s rpc call [%s:%s:%s] time out", GlobalCfg.Namespace, serviceName, rpcName)}
            ch <- []interface{}{timeout, nil}
        }
        AddCallBack(rid, []interface{}{ch}, 33, timeoutCb)
    }

    if !MessageSend(client.conn, msg) {
        return
    }
    // DeclareHook(msg, client)

    if rid != 0 {
        // block
        rsp := <- ch

        // ??????
        error, isTimeout := rsp[0].(RpcCallTimeOutError)
        if isTimeout && reSendCnt > 0 {
            DEBUG_LOG("[s2s call sync] - [%s:%s] timeout... resend.%v", serviceName, rpcName, reSendCnt)
            return RpcCall(serviceName, rpcName, GenGid(), reSendCnt - 1, args...)
        }

        if isTimeout {
            err = error.err
        } else {
            err = rsp[0].(string)
        }

        if rsp[1] != nil {
            reply = rsp[1].(map[string]interface{})
        }

        DEBUG_LOG("[s2s call sync] - [%s:%s] args[%v] err[%v] reply[%v]", serviceName, rpcName, args, err, reply)

        return
    }

    DEBUG_LOG("[s2s call async] - [%s:%s] args[%v]", serviceName, rpcName, args)
    return
}

func RpcCallSync(serviceName string, rpcName string, args ...interface{}) (string, map[string]interface{}) {
    return RpcCall(serviceName, rpcName, GenGid(), 3, args...) // ??????????????????3???
}

func RpcCallAsync(serviceName string, rpcName string, args ...interface{}) {
    RpcCall(serviceName, rpcName, 0, 0, args...)
}

// PushUnsafe("client1", "server", "push_test", int32(10), "hi bao")
func PushUnsafe(clientID string, tpe string, rpcName string, args ...interface{}) {
    innerRpc := rpcMgr.RpcEncode(rpcName, args...)
    // msg := rpcMgr.MessageEncode(innerRpc)
    // DEBUG_LOG("---for debug %d", len(innerRpc))
    RpcCallAsync("PushService", MSG_S2P_PUSH, clientID, tpe, false, innerRpc)
}

func PushSafe(clientID string, tpe string, rpcName string, args ...interface{}) {
    innerRpc := rpcMgr.RpcEncode(rpcName, args...)
    // msg := rpcMgr.MessageEncode(innerRpc)
    RpcCallAsync("PushService", MSG_S2P_PUSH, clientID, tpe, true, innerRpc)
}

func PushServerUnsafe(clientID string, rpcName string, args ...interface{}) {
    PushUnsafe(clientID, "server", rpcName, args...)
}

func PushClientUnsafe(clientID string, rpcName string, args ...interface{}) {
    PushUnsafe(clientID, "client", rpcName, args...)
}

func PushServerSafe(clientID string, rpcName string, args ...interface{}) {
    PushSafe(clientID, "server", rpcName, args...)
}

func PushClientSafe(clientID string, rpcName string, args ...interface{}) {
    PushSafe(clientID, "client", rpcName, args...)
}
