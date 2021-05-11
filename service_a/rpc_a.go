package main


import (
	msf "ms_framework"
)


type RpcAReq struct {
	I 	uint32
	F 	float32
	S 	string
	M   map[string]interface{}
	L   []int32
}

type RpcARsp struct {
	Success 	bool
	Req 		uint32
}

type RpcAHandler struct {
	req 	RpcAReq
	rsp 	RpcARsp
}

func (r *RpcAHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcAHandler) GetRspPtr() interface{} {return &(r.rsp)}

func (r *RpcAHandler) Process(session *msf.Session) {
	r.rsp.Success = true
	r.rsp.Req = r.req.I
	// msf.DEBUG_LOG("RpcAHandler %+v", *r)

	msf.RpcCallSync("ServiceB", "rpc_b", uint32(10))
}
