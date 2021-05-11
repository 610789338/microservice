package ms_framework

import (
	"net"
)


type RpcS2GRpcRsp struct {
	GRid	uint32
	Error   string
	Reply  	map[string]interface{}
}

type RpcS2GRpcRspHandler struct {
	req 	RpcS2GRpcRsp
}

func (r *RpcS2GRpcRspHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcS2GRpcRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcS2GRpcRspHandler) Process(session *Session) {
	// 根据GRid找到clientID:Rid，将GRid替换成Rid，然后把Error和Reply透传给client
	DEBUG_LOG("RpcS2GRpcRspHandler: %+v", r.req)

	cbs := GetCallBack(r.req.GRid)
	if nil == cbs {
		return
	}

	rid, connID := cbs[0].(uint32), cbs[1].(CONN_ID)

	var conn net.Conn
	if client := GetClient(connID); client != nil {
		conn = client.conn
	} else if remote := GetRemote(connID); remote != nil {
		conn = remote.conn
	}

	if nil == conn {
		ERROR_LOG("[RpcS2GRpcRspHandler] connID %s not exist", connID)
		return
	}

	rpc := RpcEncode(MSG_G2C_RPC_RSP, rid, r.req.Error, r.req.Reply)
	msg := MessageEncode(rpc)

	wLen, err := conn.Write(msg)
	if err != nil {
		ERROR_LOG("write %v error %v", conn.RemoteAddr(), err)
	}

	if wLen != len(msg) {
		WARN_LOG("write len(%v) != msg len(%v) @%v", wLen, len(msg), conn.RemoteAddr())
	}
}
