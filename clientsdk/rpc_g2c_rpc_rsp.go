package clientsdk


import (
	msf "ms_framework"
)


type RpcG2CRpcRspReq struct {
	Rid 	uint32
	Error   string
	Reply   map[string]interface{}
}

type RpcG2CRpcRspHandler struct {
	req 	RpcG2CRpcRspReq
}

func (r *RpcG2CRpcRspHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcG2CRpcRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcG2CRpcRspHandler) Process(session *msf.Session) {

	cbs := msf.GetCallBack(r.req.Rid)
	if nil == cbs {
		msf.ERROR_LOG("RpcG2CRpcRspHandler GetCallBack error %v maybe timeout", r.req.Rid)
		return
	}

	cb := cbs[0].(CallBack)
	if cb != nil {
		cb(r.req.Error, r.req.Reply)
	}
}
