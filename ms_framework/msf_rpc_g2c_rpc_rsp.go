package ms_framework


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

func (r *RpcG2CRpcRspHandler) Process(session *Session) {

	cbs := GetCallBack(r.req.Rid)
	if nil == cbs {
		ERROR_LOG("RpcG2CRpcRspHandler GetCallBack error %v", r.req.Rid)
		return
	}

	ch := cbs[0].(chan []interface{})
	ch <- []interface{}{r.req.Error, r.req.Reply}
}
