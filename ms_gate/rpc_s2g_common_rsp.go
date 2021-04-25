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

func (r *RpcS2GCommonRspHandler) Process(session *msf.Session) {
	// 根据GRid找到clientID:Rid，将GRid替换成Rid，然后把Error和Reply透传给client
	msf.DEBUG_LOG("RpcS2GCommonRspHandler: %+v", r.req)

	rcIDChan := make(chan []interface{})
	gCbChan <- []interface{}{"get&del", r.req.GRid, rcIDChan}
	
	rcID := <- rcIDChan
	if nil == rcID {
		return
	}

	rid := rcID[0].(uint32)
	clientID := rcID[1].(msf.CLIENT_ID)

	rc := msf.GetClient(clientID)
	if nil == rc {
		return
	}

	rpc := msf.RpcEncode(msf.MSG_COMMON_RSP, rid, r.req.Error, r.req.Reply)
	msg := msf.MessageEncode(rpc)

	wLen, err := rc.Write(msg)
	if err != nil {
		msf.ERROR_LOG("write %v error %v", rc.RemoteAddr(), err)
	}

	if wLen != len(msg) {
		msf.WARN_LOG("write len(%v) != msg len(%v) @%v", wLen, len(msg), rc.RemoteAddr())
	}
}
