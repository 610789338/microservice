package ms_framework

import (
	"fmt"
	"github.com/vmihailenco/msgpack"
	"bytes"
	"reflect"
)


type RpcG2SRpcCallReq struct {
	GRid 			uint32
	InnerRpc		[]byte  // 
}

type RpcG2SRpcCallHandler struct {
	req 	RpcG2SRpcCallReq
}

func (r *RpcG2SRpcCallHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcG2SRpcCallHandler) GetRspPtr() interface{} {return nil}

func (r *RpcG2SRpcCallHandler) Process(session *Session) {

	rpcFvc.Count()
	error, reply := r.rpc_handler(session)

	if r.req.GRid != 0 {

		// response
		rpc := rpcMgr.RpcEncode(MSG_S2G_RPC_RSP, r.req.GRid, error, reply)
		msg := rpcMgr.MessageEncode(rpc)
		session.SendResponse(msg)
	}

	if error != "" {
		ERROR_LOG("[RpcG2SRpcCallHandler] %v", error)
	}
}

func (r *RpcG2SRpcCallHandler) rpc_handler(session *Session) (string, map[string]interface{}){

	decoder := msgpack.NewDecoder(bytes.NewBuffer(r.req.InnerRpc))

	var rpcName string
	decoder.Decode(&rpcName)

	f, ok := rpcMgr.GetRpcHanderGenerator(rpcName)
	if !ok {
		return fmt.Sprintf("rpc %s not exist", rpcName), nil
	}

	rpc := f()
	reqPtr := reflect.ValueOf(rpc.GetReqPtr())
	if rpc.GetRspPtr() != nil && !reqPtr.IsNil() {
		stValue := reqPtr.Elem()
		for i := 0; i < stValue.NumField(); i++ {
			nv := reflect.New(stValue.Field(i).Type())
			if err := decoder.Decode(nv.Interface()); err != nil {
				return fmt.Sprint("rpc %s arg %s(%v) decode error: %v", rpcName, stValue.Type().Field(i).Name, nv.Type(), err), nil
			}

			stValue.Field(i).Set(nv.Elem())
		}
	}

	rpc.Process(session)

	if r.req.GRid != 0 {
		rspPtr := reflect.ValueOf(rpc.GetRspPtr())
		if rpc.GetRspPtr() == nil || rspPtr.IsNil() {
			if r.req.GRid != 0 {
				return fmt.Sprint("rpc %s need response but get nil", rpcName), nil
			}
			return "", nil
		}

		// for response
		stMap := make(map[string]interface{})
		stValue := rspPtr.Elem()
		for i := 0; i < stValue.NumField(); i++ {
			stMap[stValue.Type().Field(i).Name] = stValue.Field(i).Interface()
		}

		DEBUG_LOG("[RpcG2SRpcCallHandler] - METHOD[%s] args[%v] response[%v]", rpcName, rpc.GetReqPtr(), stMap)
		return "", stMap
	}

	DEBUG_LOG("[RpcG2SRpcCallHandler] - METHOD[%s] args[%v] response[nil]", rpcName, rpc.GetReqPtr())
	return "", nil
}
