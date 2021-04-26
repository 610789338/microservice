package clientsdk


import (
	msf "ms_framework"
)


type RpcCommonRspReq struct {
	Rid 	uint32
	Error   string
	Reply   map[string]interface{}
}

type RpcCommonRspHandler struct {
	req 	RpcCommonRspReq
}

func (r *RpcCommonRspHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcCommonRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcCommonRspHandler) Process(session *msf.Session) {
	// msf.DEBUG_LOG("[RpcCommonRspHandler] rid(%v) error(%v) reply(%v)", r.req.Rid, r.req.Error, r.req.Reply)

	cb := GetCallBack(r.req.Rid)
	if cb != nil {
		cb(r.req.Error, r.req.Reply)
	}
}
