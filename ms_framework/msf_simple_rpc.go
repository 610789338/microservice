package ms_framework

import (
	// "errors"
	"reflect"
	"github.com/vmihailenco/msgpack"
	"bytes"
	// "fmt"
)


var MAX_PACKET_SIZE uint32 = 16*1024  // 16K
var MESSAGE_SIZE_LEN uint32 = 4
var RID_LEN uint32 = 4

var MSG_C2G_RPC_ROUTE 	= "a"  // client to gate rpc route
var MSG_G2S_RPC_CALL 	= "b"  // gate to service rpc call
var MSG_COMMON_RSP 		= "c"  // common response include s2g && g2c

type encodeWithoutFieldName interface{
	EncodeWithoutFieldName ()
}

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
	Process(c *TcpClient)
}

type RpcHanderGenerator func() RpcHandler

type SimpleRpcMgr struct {
	rpcs 	map[string]RpcHanderGenerator
}

func (rmgr *SimpleRpcMgr) RegistRpcHandler(name string, gen RpcHanderGenerator) {
	rpc := gen()
	ptrValue := reflect.ValueOf(rpc.GetReqPtr())
	if ptrValue.Kind() != reflect.Ptr {
		panic("rpc.GetReqPtr() must return a pointer")
	}

	stValue := ptrValue.Elem()
	if stValue.Kind() != reflect.Struct {
		panic("rpc.GetReqPtr() must be struct")
	}

	rmgr.rpcs[name] = gen
}

func (rmgr *SimpleRpcMgr) MessageDecode(c *TcpClient, buf []byte) (uint32, []byte) {
	var offset uint32 = 0
	var msgsRsp []byte = []byte{}

	var bufLen uint32 = uint32(len(buf))

	for offset < bufLen {
		if bufLen - offset < MESSAGE_SIZE_LEN {
			DEBUG_LOG("remain len(%d) < MESSAGE_SIZE_LEN(%d)", bufLen - offset, MESSAGE_SIZE_LEN)
			break
		}

		pkgLen := ReadPacketLen(buf[offset:])
		if bufLen - offset < MESSAGE_SIZE_LEN + pkgLen {
			DEBUG_LOG("remain len(%d) < MESSAGE_SIZE_LEN(%d) + pkgLen(%d)", bufLen - offset, MESSAGE_SIZE_LEN, pkgLen)
			break
		}

		offset += MESSAGE_SIZE_LEN

		if pkgLen > MAX_PACKET_SIZE {
			ERROR_LOG("packet size too long %d > %d", pkgLen, MAX_PACKET_SIZE)
		} else {

			rsp := rmgr.RpcDecode(c, buf[offset: offset + pkgLen])
			if rsp != nil {
				msgRsp := rmgr.MessageEncode(rsp)
				msgsRsp = append(msgsRsp, msgRsp...)
			}
		}

		offset += pkgLen
	}

	return offset, msgsRsp
}

func (rmgr *SimpleRpcMgr) RpcDecode(c *TcpClient, b []byte) []byte {

	decoder := msgpack.NewDecoder(bytes.NewBuffer(b))

	var rpcName string
	decoder.Decode(&rpcName)

	f, ok := rmgr.rpcs[rpcName]
	if !ok {
		ERROR_LOG("rpc %s not exist", rpcName)
		return nil
	}

	rpc := f()
	reqPtr := reflect.ValueOf(rpc.GetReqPtr())
	stValue := reqPtr.Elem()
	for i := 0; i < stValue.NumField(); i++ {
		nv := reflect.New(stValue.Field(i).Type())
		if err := decoder.Decode(nv.Interface()); err != nil {
			ERROR_LOG("rpc(%s) arg(%s-%v) decode error: %v", rpcName, stValue.Type().Field(i).Name, nv.Type(), err)
			return nil
		}

		stValue.Field(i).Set(nv.Elem())
	}

	rpc.Process(c)

	rspPtr := reflect.ValueOf(rpc.GetRspPtr())
	if rpc.GetRspPtr() == nil || rspPtr.IsNil() {
		return nil
	}

	// for response
	args := []interface{}{}
	switch rpc.GetRspPtr().(type) {
	case encodeWithoutFieldName:
		stValue = rspPtr.Elem()
		for i := 0; i < stValue.NumField(); i++ {
			args = append(args, stValue.Field(i).Interface())
		}

	default:
		stMap := make(map[string]interface{})
		stValue = rspPtr.Elem()
		for i := 0; i < stValue.NumField(); i++ {
			stMap[stValue.Type().Field(i).Name] = stValue.Field(i).Interface()
		}

		args = append(args, stMap)
	}

	return rmgr.RpcEncode(MSG_COMMON_RSP, args...)
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
	rpcMgr.RegistRpcHandler(MSG_G2S_RPC_CALL, func() RpcHandler {return new(RpcG2SRpcCallHandler)})
}

func RegistRpcHandler(name string, gen RpcHanderGenerator) {
	rpcMgr.RegistRpcHandler(name, gen)
}

func MessageEncode(b []byte) []byte {
	return rpcMgr.MessageEncode(b)
}

func RpcEncode(name string, args ...interface{}) []byte {
	return rpcMgr.RpcEncode(name, args...)
}
