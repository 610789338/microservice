package main

import (
	msf "ms_framework"
	"fmt"
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
func (*RpcC2GRpcRouteRsp) EncodeWithoutFieldName(){}

type RpcC2GRpcRouteHandler struct {
	req 	RpcC2GRpcRouteReq
	rsp 	*RpcC2GRpcRouteRsp
}

func (r *RpcC2GRpcRouteHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcC2GRpcRouteHandler) GetRspPtr() interface{} {return r.rsp}

func (r *RpcC2GRpcRouteHandler) Process(c *msf.TcpClient) {
	// 消息路由，根据namespace:service:method从本地ip缓存中找到tcp连接，然后将消息路由过去

	// * 从B里面解析出Rid, if Rid != 0
	// * 生成GRid，并建立GRid <-> clientID:Rid的对应关系
	// * 用GRid替换掉rpc中的rid

	remoteID := msf.GetRemoteID(r.req.NameSpace, r.req.Service)
	remote := msf.ChoiceRemote(remoteID)

	if remote != nil {
		grid := uint32(0)
		if r.req.Rid != 0 {
			grid = msf.GenGid()
		}

		rpc := msf.RpcEncode(msf.MSG_G2S_RPC_CALL, grid, r.req.InnerRpc)
		msg := msf.MessageEncode(rpc)

		wLen, err := remote.Write(msg)
		if err != nil {
			msf.ERROR_LOG("write %v error %v", remote.RemoteAddr(), err)
			return
		}

		if wLen != len(msg) {
			msf.WARN_LOG("write len(%v) != msg len(%v) @%v", wLen, len(msg), remote.RemoteAddr())
		}

		gCbChan <- []interface{}{"add", grid, r.req.Rid, c.GetClientID()}

	} else {
		// error response
		r.rsp = &RpcC2GRpcRouteRsp{Rid: r.req.Rid, Error: fmt.Sprintf("service[%s:%s] not exist", r.req.NameSpace, r.req.Service), Reply: nil}
	}

	if r.rsp != nil {
		msf.ERROR_LOG("[RpcC2GRpcRouteHandler] - SERVICE - [%s:%s] rid[%v] response[%v]", r.req.NameSpace, r.req.Service, r.req.Rid, r.rsp.Error)
	} else {
		msf.DEBUG_LOG("[RpcC2GRpcRouteHandler] - SERVICE - [%s:%s] rid[%v] response[nil]", r.req.NameSpace, r.req.Service, r.req.Rid)
	}
}
