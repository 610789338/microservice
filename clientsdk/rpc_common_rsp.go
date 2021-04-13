package clientsdk


import (
	msf "ms_framework"
)


type RpcCommonRspReq struct {
	// Rid 	uint32
	// Error   string
	Reply   map[string]interface{}
}

type RpcCommonRspHandler struct {
	req 	RpcCommonRspReq
}

func (r *RpcCommonRspHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcCommonRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcCommonRspHandler) Process() {
	msf.INFO_LOG("RpcCommonRspHandler %+v", *r)

	// cb, ok := gCbMap[r.req.Rid]
	// if !ok {
	// 	msf.ERROR_LOG("rid %v not exsit", r.req.Rid)
	// 	return
	// }

	// cb(r.req.Error, r.req.Reply)
	// gCbChan <- []interface{}{"del", r.req.Rid}
}
