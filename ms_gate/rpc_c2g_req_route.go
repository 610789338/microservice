package main

import (
	msf "ms_framework"
)


type RpcC2GRpcRouteReq struct {
	NameSpace	 	string
	Service 		string
	Rid 			uint32
	InnerRpc		[]byte
}

type RpcC2GRpcRouteRsp struct {
	Rid 			uint32
	Error 			string
	Reply   		map[string]interface{}
}

type RpcC2GRpcRouteHandler struct {
	req 	RpcC2GRpcRouteReq
	rsp 	*RpcC2GRpcRouteRsp
}

func (r *RpcC2GRpcRouteHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcC2GRpcRouteHandler) GetRspPtr() interface{} {return r.rsp}

func (r *RpcC2GRpcRouteHandler) Process() {
	// 消息路由，根据namespace:service:method从本地ip缓存中找到tcp连接，然后将消息路由过去
	// TODO LIST
	// * 建立本地路由缓存
	// * 本地路由缓存更新：主动更新 and 被动更新（依赖etcd）
	// * 负载均衡

	// * 从B里面解析出Rid, if Rid != 0
	// * 生成GRid，并建立GRid <-> clientID:Rid的对应关系
	// * 用GRid替换掉rpc中的rid

	msf.INFO_LOG("RpcC2GRpcRouteHandler: %v", r.req)

	remoteID := msf.GetRemoteID(r.req.NameSpace, r.req.Service)
	remote := msf.ChoiceRemote(remoteID)

	if nil == remote {
		msf.ERROR_LOG("")
	}

	// error response
	r.rsp = &RpcC2GRpcRouteRsp{Rid: r.req.Rid, Error: "service not exist", Reply: nil}
}
