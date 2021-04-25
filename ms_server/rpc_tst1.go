package main


import (
	msf "ms_framework"
)


type RpcTest1Req struct {
	I 	uint32
}

type RpcTest1Rsp struct {
	Success 	bool
	Req 		uint32
}

type RpcTest1Handler struct {
	req 	RpcTest1Req
	rsp 	RpcTest1Rsp
}

func (r *RpcTest1Handler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcTest1Handler) GetRspPtr() interface{} {return &(r.rsp)}

func (r *RpcTest1Handler) Process(session *msf.Session) {
	r.rsp.Success = true
	r.rsp.Req = r.req.I
	msf.INFO_LOG("RpcTest1Handler %+v", *r)
}
