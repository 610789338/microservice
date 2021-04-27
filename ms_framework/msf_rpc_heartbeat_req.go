package ms_framework


type RpcHeartBeatReqHandler struct {
}

func (r *RpcHeartBeatReqHandler) GetReqPtr() interface{} {return nil}
func (r *RpcHeartBeatReqHandler) GetRspPtr() interface{} {return nil}

func (r *RpcHeartBeatReqHandler) Process(session *Session) {
	DEBUG_LOG("heart beat request from %v", GetConnID(session.conn))

	rpc := RpcEncode(MSG_HEART_BEAT_RSP)
	msg := MessageEncode(rpc)
	session.SendResponse(msg)
}
