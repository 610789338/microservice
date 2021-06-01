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
	SessionGateProxy
)

type Session struct {
	typ    		int8
	id     		string
	conn		net.Conn
}

func (session *Session) GetID() string {
	return session.id
}

func CreateSession(typ int8, id string, conn net.Conn) *Session {
	return &Session{typ: typ, id: id, conn: conn}
}


var MAX_PACKET_SIZE uint32 = 16*1024  // 16K
var MESSAGE_SIZE_LEN uint32 = 4
var RID_LEN uint32 = 4

func ReadPacketLen(buf []byte) uint32 {
	return ReadUint32(buf)
}

func WritePacketLen(buf []byte, v uint32) {
	WriteUint32(buf, v)
}

func ReadRid(buf []byte) uint32 {
	return ReadUint32(buf)
}

func WriteRid(buf []byte, v uint32) {
	WriteUint32(buf, v)
}

type RpcHandler interface {
	GetReqPtr() interface{}
	GetRspPtr() interface{}
	Process(session *Session)
}

type RpcHanderGenerator func() RpcHandler

type WithErrRsp interface {
	SetErr(err string)
	GetErr() string
}

type ErrRsp struct {
	err 	string
}

func (r *ErrRsp) SetErr(err string) {r.err = err}
func (r *ErrRsp) GetErr() string {return r.err}

func GetResponseErr(i interface{}) string {
	v, ok := i.(WithErrRsp)	
	// DEBUG_LOG("### GetResponseErr %T isok? %v - err: %v", i, ok, i)

	if ok {
		return v.GetErr()
	} else {
		return ""
	}
}

type SimpleRpcMgr struct {
	rpcs 	map[string]RpcHanderGenerator
}

func (rmgr *SimpleRpcMgr) RegistRpcHandler(name string, gen RpcHanderGenerator) {
	_, ok := rmgr.rpcs[name]
	if ok {
		panic(fmt.Sprintf("RegistRpcHandler %s repeat !!!", name))
	}

	rmgr.rpcs[name] = gen
}

func (rmgr *SimpleRpcMgr) RegistRpcHandlerForce(name string, gen RpcHanderGenerator) {
	rmgr.rpcs[name] = gen
}

func (rmgr *SimpleRpcMgr) MessageDecode(session *Session, msg []byte) uint32 {
	var offset uint32 = 0

	var bufLen uint32 = uint32(len(msg))

	for offset < bufLen {
		if bufLen - offset < MESSAGE_SIZE_LEN {
			DEBUG_LOG("remain len(%d) < MESSAGE_SIZE_LEN(%d)", bufLen - offset, MESSAGE_SIZE_LEN)
			break
		}

		pkgLen := ReadPacketLen(msg[offset:])
		if bufLen - offset < MESSAGE_SIZE_LEN + pkgLen {
			DEBUG_LOG("remain len(%d) < MESSAGE_SIZE_LEN(%d) + pkgLen(%d)", bufLen - offset, MESSAGE_SIZE_LEN, pkgLen)
			break
		}

		offset += MESSAGE_SIZE_LEN

		if pkgLen > MAX_PACKET_SIZE {
			ERROR_LOG("packet size too long %d > %d", pkgLen, MAX_PACKET_SIZE)
		} else {
			buf := make([]byte, pkgLen)
			copy(buf, msg[offset: offset + pkgLen])
			go gTaskPool.ProduceTask(session, buf)

			// go rmgr.RpcDecode(session, buf)
			// rmgr.RpcDecode(session, buf)
			// rmgr.RpcDecode(session, msg[offset: offset + pkgLen])
		}

		offset += pkgLen
	}

	return offset
}

func (rmgr *SimpleRpcMgr) RpcDecode(session *Session, buf []byte) {

	decoder := msgpack.NewDecoder(bytes.NewBuffer(buf))

	var rpcName string
	decoder.Decode(&rpcName)

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

func (rmgr *SimpleRpcMgr) MessageEncode(buf []byte) []byte {

	bufLen := uint32(len(buf))
	msg := make([]byte, MESSAGE_SIZE_LEN + bufLen)
	WritePacketLen(msg, bufLen)
	// WriteRid(msg[MESSAGE_SIZE_LEN:], rid)
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
	rpcMgr = &SimpleRpcMgr{rpcs: make(map[string]RpcHanderGenerator)}

	// default handler
	rpcMgr.RegistRpcHandler(MSG_C2G_RPC_ROUTE, 			func() RpcHandler {return new(RpcC2GRpcRouteHandler)}) 	// for gate
	rpcMgr.RegistRpcHandler(MSG_G2S_RPC_CALL, 			func() RpcHandler {return new(RpcG2SRpcCallHandler)})  	// for service
	rpcMgr.RegistRpcHandler(MSG_S2G_RPC_RSP, 			func() RpcHandler {return new(RpcS2GRpcRspHandler)})   	// for gate
	rpcMgr.RegistRpcHandler(MSG_G2C_RPC_RSP, 			func() RpcHandler {return new(RpcG2CRpcRspHandler)}) 	// for client


	rpcMgr.RegistRpcHandler(MSG_HEART_BEAT_REQ, 		func() RpcHandler {return new(RpcHeartBeatReqHandler)}) // for all
	rpcMgr.RegistRpcHandler(MSG_HEART_BEAT_RSP,			func() RpcHandler {return new(RpcHeartBeatRspHandler)}) // for all

	rpcMgr.RegistRpcHandler(MSG_G2S_IDENTITY_REPORT,	func() RpcHandler {return new(RpcG2SIdentityReportHandler)}) // for service
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
	err 	string
}

func RpcCall(serviceName string, rpcName string, rid uint32, reSendCnt int8, args ...interface{}) (err string, reply map[string]interface{}) {

	innerRpc := rpcMgr.RpcEncode(rpcName, args...)
	rpc := rpcMgr.RpcEncode(MSG_C2G_RPC_ROUTE, GlobalCfg.Namespace, serviceName, rid, innerRpc)
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

		// 重发
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
	return RpcCall(serviceName, rpcName, GenGid(), 3, args...) // 默认超时重发3次
}

func RpcCallAsync(serviceName string, rpcName string, args ...interface{}) {
	RpcCall(serviceName, rpcName, 0, 0, args...)
}
