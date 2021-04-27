package ms_framework


type RpcHeartBeatRspHandler struct {
}

func (r *RpcHeartBeatRspHandler) GetReqPtr() interface{} {return nil}
func (r *RpcHeartBeatRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcHeartBeatRspHandler) Process(session *Session) {
	DEBUG_LOG("heart beat response from %v", GetConnID(session.conn))
}
