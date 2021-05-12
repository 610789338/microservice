package ms_framework

import (
	"fmt"
)

type RpcC2GRpcRouteReq struct {
	NameSpace	 	string
	Service 		string
	Rid 			uint32
	InnerRpc		[]byte
}

type RpcC2GRpcRouteHandler struct {
	req 	RpcC2GRpcRouteReq
}

func (r *RpcC2GRpcRouteHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcC2GRpcRouteHandler) GetRspPtr() interface{} {return nil}

func (r *RpcC2GRpcRouteHandler) Process(session *Session) {
	/*
	 * 消息路由，根据namespace:service:method从本地ip缓存中找到对应service的tcp连接，然后将消息路由过去
	 * 从B里面解析出Rid
	 * if Rid != 0
	 *   生成GRid，并建立GRid <-> clientID:Rid的对应关系
	 * 往service发送MSG_G2S_RPC_CALL请求
	 */

	rpcFvc.Count()
	
	remoteID := GetRemoteID(r.req.NameSpace, r.req.Service)
	remote := ChoiceRemote(remoteID)

	if remote != nil {

		DEBUG_LOG("[RpcC2GRpcRouteHandler] - SERVICE - [%s:%s] rid[%v] response[nil]", r.req.NameSpace, r.req.Service, r.req.Rid)

		grid := uint32(0)
		if r.req.Rid != 0 {
			grid = GenGid()
		}

		if r.req.Rid != 0 {
			// must before remote write
			AddCallBack(grid, []interface{}{r.req.Rid, CONN_ID(session.GetID())})
		}

		rpc := RpcEncode(MSG_G2S_RPC_CALL, grid, r.req.InnerRpc)
		msg := MessageEncode(rpc)

		wLen, err := remote.Write(msg)
		if err != nil {
			ERROR_LOG("write %v error %v", remote.RemoteAddr(), err)
			return
		}

		if wLen != len(msg) {
			WARN_LOG("write len(%v) != msg len(%v) @%v", wLen, len(msg), remote.RemoteAddr())
		}

	} else {
		
		error := fmt.Sprintf("service %s:%s not exist", r.req.NameSpace, r.req.Service)
		DEBUG_LOG("[RpcC2GRpcRouteHandler] - SERVICE - [%s:%s] rid[%v] response[%v]", r.req.NameSpace, r.req.Service, r.req.Rid, error)

		// error response
		rpc := rpcMgr.RpcEncode(MSG_G2C_RPC_RSP, r.req.Rid, error, nil)
		msg := rpcMgr.MessageEncode(rpc)
		session.SendResponse(msg)
	}
}
