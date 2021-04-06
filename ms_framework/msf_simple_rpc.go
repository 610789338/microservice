package ms_framework

import (
	// "errors"
	"reflect"
	"github.com/vmihailenco/msgpack"
	"bytes"
)


var MAX_PACKET_SIZE uint32 = 16*1024  // 16K
var PACKAGE_SIZE_LEN uint32 = 4

func ReadPacketLen(buf []byte) uint32 {
	return (uint32(buf[0]) << 24) |
		(uint32(buf[1]) << 16) |
		(uint32(buf[2]) << 8) |
		uint32(buf[3])
}

func WritePacketLen(buf []byte, v uint32) {
	buf[0] = byte(v & 0xFF000000)
	buf[1] = byte(v & 0xFF0000)
	buf[2] = byte(v & 0xFF00)
	buf[3] = byte(v & 0xFF)
}

type SimpleRpcMgr struct {
	rpcs 	map[string]RpcHandler
}

func (rmgr *SimpleRpcMgr) RegistRpcHandler(name string, rpc RpcHandler) {
	ptrValue := reflect.ValueOf(rpc.GetReqPtr())
	if ptrValue.Kind() != reflect.Ptr {
		panic("rpc.GetReqPtr() must return a pointer")
	}

	stValue := ptrValue.Elem()
	if stValue.Kind() != reflect.Struct {
		panic("rpc.GetReqPtr() must be struct")
	}

	rmgr.rpcs[name] = rpc
}

func (rmgr *SimpleRpcMgr) RpcParse(buf []byte) uint32 {
	var remainLen uint32 = uint32(len(buf))
	var offset uint32 = 0

	for remainLen > 0 {
		if remainLen < PACKAGE_SIZE_LEN {
			DEBUG_LOG("len(%d) - offset(%d) < PACKAGE_SIZE_LEN(%d)", remainLen, offset, PACKAGE_SIZE_LEN)
			break
		}

		pkgLen := ReadPacketLen(buf[offset:])
		if remainLen < PACKAGE_SIZE_LEN + pkgLen {
			DEBUG_LOG("len(%d) < PACKAGE_SIZE_LEN(%d) + pkgLen(%d)", remainLen, PACKAGE_SIZE_LEN, pkgLen)
			break
		}

		offset += PACKAGE_SIZE_LEN
		remainLen -= PACKAGE_SIZE_LEN

		if pkgLen > MAX_PACKET_SIZE {
			ERROR_LOG("packet size too long %d > %d", pkgLen, MAX_PACKET_SIZE)
		} else {
			rmgr.RpcDecode(buf[offset: offset + pkgLen])
		}

		offset += pkgLen
		remainLen -= pkgLen
	}

	return offset
}

func (rmgr *SimpleRpcMgr) RpcDecode(b []byte) {

	decoder := msgpack.NewDecoder(bytes.NewBuffer(b))

	var rpcName string
	decoder.Decode(&rpcName)

	rpc, ok := rmgr.rpcs[rpcName]
	if !ok {
		ERROR_LOG("rpc %s not exist", rpcName)
		return
	}

	ptrValue := reflect.ValueOf(rpc.GetReqPtr())
	stValue := ptrValue.Elem()
	for i := 0; i < stValue.NumField(); i++ {
		nv := reflect.New(stValue.Field(i).Type())
		if err := decoder.Decode(nv.Interface()); err != nil {
			ERROR_LOG("rpc(%s) arg(%s) decode error: %v", rpcName, stValue.Type().Field(i).Name, err)
			return
		}

		stValue.Field(i).Set(nv.Elem())
	}

	rpc.Process()

	// TODO: response
	// rpc.GetRspPtr()
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

// func (rmgr *SimpleRpcMgr) RpcEncode(name string, req interface{}) []byte {

// 	writer := &bytes.Buffer{}
// 	encoder := msgpack.NewEncoder(writer)
// 	if err := encoder.Encode(name); err != nil {
// 		ERROR_LOG("encode rpc name error %v", err)
// 	}

// 	ptrValue := reflect.ValueOf(req)
// 	stValue := ptrValue.Elem()
// 	for i := 0; i < stValue.NumField(); i++ {
// 		if err := encoder.Encode(stValue.Field(i).Interface()); err != nil {
// 			ERROR_LOG("args encode error %s - %s: %v", name, stValue.Field(i).Type().Name(), err)
// 			continue
// 		}
// 	}

// 	return writer.Bytes()
// }

func CreateSimpleRpcMgr() RpcMgr {
	return &SimpleRpcMgr{rpcs: make(map[string]RpcHandler)}
}
