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

func (session *Session) SendResponse(msg []byte) {
	if len(msg) != 0 {
		wLen, err := session.conn.Write(msg)
		if err != nil {
			ERROR_LOG("write %v error %v", session.conn.RemoteAddr(), err)
		}

		if wLen != len(msg) {
			WARN_LOG("write len(%v) != rsp msg len(%v) @%v", wLen, len(msg), session.conn.RemoteAddr())
		}
	}
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

// client include mservice/gameserver/gameclient
var MSG_C2G_RPC_ROUTE 			= "a"  // client to gate rpc route
var MSG_G2S_RPC_CALL 			= "b"  // gate to service rpc call
var MSG_S2G_RPC_RSP 			= "c"  // service to gate rpc response
var MSG_G2C_RPC_RSP 			= "d"  // gate to client rpc response

var MSG_HEART_BEAT_REQ 			= "e"  // heart beat request
var MSG_HEART_BEAT_RSP 			= "f"  // heart beat response


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
			go rmgr.RpcDecode(session, buf)
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

	handlerGen, ok := rmgr.rpcs[rpcName]
	if !ok {
		ERROR_LOG("rpc %s not exist", rpcName)
		return
	}

	rpcHandler := handlerGen()
	if rpcHandler.GetReqPtr() != nil {
		reqPtr := reflect.ValueOf(rpcHandler.GetReqPtr())
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

	rpcHandler.Process(session)

	// rspPtr := reflect.ValueOf(rpcHandler.GetRspPtr())
	// if rpcHandler.GetRspPtr() == nil || rspPtr.IsNil() {
	// 	return
	// }
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

	for _, arg := range(args) {
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
	rpcMgr.RegistRpcHandler(MSG_C2G_RPC_ROUTE, 		func() RpcHandler {return new(RpcC2GRpcRouteHandler)}) 	// for gate
	rpcMgr.RegistRpcHandler(MSG_G2S_RPC_CALL, 		func() RpcHandler {return new(RpcG2SRpcCallHandler)})  	// for service
	rpcMgr.RegistRpcHandler(MSG_S2G_RPC_RSP, 		func() RpcHandler {return new(RpcS2GRpcRspHandler)})   	// for gate
	rpcMgr.RegistRpcHandler(MSG_G2C_RPC_RSP, 		func() RpcHandler {return new(RpcG2CRpcRspHandler)}) 	// for client


	rpcMgr.RegistRpcHandler(MSG_HEART_BEAT_REQ, 	func() RpcHandler {return new(RpcHeartBeatReqHandler)}) // for all
	rpcMgr.RegistRpcHandler(MSG_HEART_BEAT_RSP,		func() RpcHandler {return new(RpcHeartBeatRspHandler)}) // for all
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
