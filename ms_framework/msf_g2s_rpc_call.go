package ms_framework

import (
	// msf "ms_framework"
)


type RpcG2SRpcCallReq struct {
	Rid 			uint32
	Method 			string
	Args			[]byte
}

type RpcG2SRpcCallRsp struct {
	Rid 			uint32
	Error 			string
	Reply   		map[string]interface{}
}

type RpcG2SRpcCallHandler struct {
	req 	RpcG2SRpcCallReq
	rsp 	*RpcG2SRpcCallRsp
}

func (r *RpcG2SRpcCallHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcG2SRpcCallHandler) GetRspPtr() interface{} {return r.rsp}

func (r *RpcG2SRpcCallHandler) Process() {
	// TODO
	INFO_LOG("RpcG2SRpcCallHandler: %v", r.req)
}
