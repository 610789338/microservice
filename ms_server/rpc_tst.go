package main


import (
	msf "ms_framework"
)


type RpcTestReq struct {
	I 	uint32
	F 	float32
	S 	string
	M   map[string]interface{}
	L   []int32
}

type RpcTestRsp struct {
	Success 	bool
	Req 		uint32
}

type RpcTestHandler struct {
	req 	RpcTestReq
	rsp 	RpcTestRsp
}

func (r *RpcTestHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcTestHandler) GetRspPtr() interface{} {return &(r.rsp)}

func (r *RpcTestHandler) Process(session *msf.Session) {
	r.rsp.Success = true
	r.rsp.Req = r.req.I
	msf.DEBUG_LOG("RpcTestHandler %+v", *r)
}
