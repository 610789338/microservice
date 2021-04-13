package main

import (
	msf "ms_framework"
)


type RpcS2GCommonRsp struct {
	GRid	uint32
	Error   string
	Reply  	map[string]interface{}
}

type RpcS2GCommonRspHandler struct {
	req 	RpcS2GCommonRsp
}

func (r *RpcS2GCommonRspHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcS2GCommonRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcS2GCommonRspHandler) Process() {
	// 响应路由，根据GRid找到clientID:Rid，将GRid替换成Rid，然后把Error和Reply透传给client
	msf.INFO_LOG("rsp route: %v", r.req)
}
