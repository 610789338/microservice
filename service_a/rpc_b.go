package main


import (
    msf "ms_framework"
)


type RpcBReq struct {
    I           uint32
}

type RpcBRsp struct {
    Success     bool
    Req         uint32
}

type RpcBHandler struct {
    req     RpcBReq
    rsp     RpcBRsp
}

func (r *RpcBHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcBHandler) GetRspPtr() interface{} {return &(r.rsp)}
func (r *RpcBHandler) ClientAccess() bool {return true}

func (r *RpcBHandler) Process(session *msf.Session) {
    r.rsp.Success = true
    r.rsp.Req = r.req.I
    // msf.DEBUG_LOG("RpcBHandler %+v", *r)
}
